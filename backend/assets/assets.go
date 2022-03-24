package assets

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dkumor/revhttpfs"
	"github.com/rakyll/statik/fs"
	"github.com/sirupsen/logrus"
	"github.com/spf13/afero"
)

// The cached builtin Assets
var builtinAssets afero.Fs

// BuiltinAssets prepares the assets built into the executable. If no such assets are found, it tries
// to find an assets folder, and use that, which is useful for debugging and development.
func BuiltinAssets() afero.Fs {
	if builtinAssets == nil {
		statikFS, err := fs.New()
		if err != nil {
			// Try to find an assets folder in the ancestors
			cwd, err := os.Getwd()
			if err != nil {
				panic(err)
			}
			assetPath := filepath.Join(cwd, "assets")
			_, err = os.Stat(filepath.Join(assetPath, "heedy.conf"))
			for os.IsNotExist(err) {
				cwdnew := filepath.Dir(cwd)
				if cwdnew == cwd {
					panic(errors.New("Could not find assets folder"))
				}
				cwd = cwdnew
				assetPath = filepath.Join(cwd, "assets")
				_, err = os.Stat(filepath.Join(assetPath, "heedy.conf"))
			}

			logrus.Warnf("This is a debug build of heedy: using assets from %s", assetPath)

			builtinAssets = afero.NewBasePathFs(afero.NewOsFs(), assetPath)
		} else {
			builtinAssets = revhttpfs.NewReverseHttpFs(statikFS)
		}

	}

	return builtinAssets
}

// Assets holds the information that comes from loading the database folder,
// merging it with the built-in assets, and combining
type Assets struct {

	// FolderPath is the path where the database is installed.
	// it can be "" if we are running heedy in setup mode,
	// in which case it runs solely on builtin assets
	// This is the only thing that needs to be manually initialized.
	FolderPath string

	// An override to the configuration. It is merged on top of the root configuration
	// before any special processing
	ConfigOverride *Configuration

	// The active configuration. This is loaded automatically
	Config *Configuration

	// The overlay stack. index 0 represents built-in assets. Each index is just that stack element.
	Stack []afero.Fs

	// The overlay filesystems that include the builtin assets, as well as all
	// overrides from active plugins, and user overrides. It is loaded automatically
	FS afero.Fs

	// LogFile used for logging (if custom)
	LogFile *os.File
}

// Reload the assets from scratch
func (a *Assets) Reload() error {

	assetStack := make([]afero.Fs, 1)

	builtinAssets := BuiltinAssets()
	assetStack[0] = builtinAssets

	// First, we load the configuration from the builtin assets
	baseConfigBytes, err := afero.ReadFile(builtinAssets, "/heedy.conf")
	if err != nil {
		return err
	}
	baseConfiguration, err := LoadConfigBytes(baseConfigBytes, "heedy.conf")
	if err != nil {
		return err
	}

	// Some plugins come built-in. Check for the built-in plugins
	if baseConfiguration.ActivePlugins != nil {
		for _, v := range *baseConfiguration.ActivePlugins {
			_, ok := baseConfiguration.Plugins[v]
			if !ok {
				return fmt.Errorf("Builtin configuration does not define plugin '%s'", v)
			}
		}
	}

	// Next, we initialize the filesystem overlays from the builtin assets
	FS := builtinAssets

	mergedConfiguration := baseConfiguration

	if a.FolderPath == "" {
		// If there is no folder path, we are running purely on built-in assets.
		//log.Debug("No asset folder specified - running on builtin assets.")
		if a.ConfigOverride != nil {
			mergedConfiguration = MergeConfig(mergedConfiguration, a.ConfigOverride)
		}

	} else {
		// Make sure the folder path is absolute
		a.FolderPath, err = filepath.Abs(a.FolderPath)
		if err != nil {
			return err
		}

		// The os filesystem
		osfs := afero.NewOsFs()

		// First, we load the root config file, which will specify which plugins to activate
		configPath := path.Join(a.FolderPath, "heedy.conf")
		rootConfiguration, err := LoadConfigFile(configPath)
		if err != nil {
			return err
		}

		if a.ConfigOverride != nil {
			rootConfiguration = MergeConfig(rootConfiguration, a.ConfigOverride)
		}

		// Next, we go through the plugin folder one by one, and add the active plugins to configuration
		// and overlay the plugin's filesystem over assets
		if rootConfiguration.ActivePlugins != nil {

			for _, pluginName := range *rootConfiguration.ActivePlugins {
				if !strings.HasPrefix(pluginName, "-") {
					if strings.HasPrefix(pluginName, "+") {
						pluginName = pluginName[1:len(pluginName)]
					}

					pluginFolder := path.Join(a.FolderPath, "plugins", pluginName)
					pluginFolderStats, err := os.Stat(pluginFolder)
					if err != nil {
						return err
					}
					if !pluginFolderStats.IsDir() {
						return fmt.Errorf("Could not find plugin %s at %s: not a directory", pluginName, pluginFolder)
					}

					configPath := path.Join(pluginFolder, "heedy.conf")
					pluginConfiguration, err := LoadConfigFile(configPath)
					if err != nil {
						return err
					}
					mergedConfiguration = MergeConfig(mergedConfiguration, pluginConfiguration)

					pluginFs := afero.NewBasePathFs(osfs, pluginFolder)
					assetStack = append(assetStack, pluginFs)
					FS = afero.NewCopyOnWriteFs(FS, pluginFs)
				}
			}

		}

		// Finally, we overlay the root directory and root config
		mergedConfiguration = MergeConfig(mergedConfiguration, rootConfiguration)
		mainFs := afero.NewBasePathFs(osfs, a.FolderPath)
		assetStack = append(assetStack, mainFs)
		FS = afero.NewCopyOnWriteFs(FS, mainFs)

		// Get the full list of active plugins here
		mergedConfiguration.ActivePlugins = MergeStringArrays(baseConfiguration.ActivePlugins, rootConfiguration.ActivePlugins)

	}

	// Next clean up the addresses
	c := mergedConfiguration

	addr, err := ParseAddress(a.DataDir(), c.GetAddr())
	if err != nil {
		return err
	}
	c.Addr = &addr
	api, err := ParseAddress(a.DataDir(), c.GetAPI())
	if err != nil {
		return err
	}
	c.API = &api

	// Finally, set the URL if it isn't set
	if c.URL == nil || *c.URL == "" {
		if strings.HasPrefix(*c.Addr, "unix:") {
			c.URL = c.Addr
		} else {
			host, port, err := net.SplitHostPort(*c.Addr)
			if err != nil {
				return err
			}
			if host == "" {
				host = GetOutboundIP()
			}
			myurl := fmt.Sprintf("http://%s:%s", host, port)
			c.URL = &myurl
		}
	}
	if strings.HasSuffix(*c.URL, "/") {
		noslash := (*c.URL)[:len(*c.URL)-1]
		c.URL = &noslash
	}

	// Set the new config and assets
	a.Config = c
	a.FS = FS
	a.Stack = assetStack

	if err := Validate(a.Config); err != nil {
		return err
	}

	if a.FolderPath != "" {
		// set the logging level based on the config, unless we're going purely from built-in assets
		if a.Config.LogLevel != nil {
			lvl, err := logrus.ParseLevel(*a.Config.LogLevel)
			if err != nil {
				return err
			}
			logrus.SetLevel(lvl)
		}
		if a.Config.LogDir != nil {
			logdir := a.LogDir()
			if logdir == "stdout" {
				logrus.SetOutput(os.Stdout)
			} else {
				if _, err = os.Stat(logdir); err != nil {
					if err = os.Mkdir(logdir, os.ModePerm); err != nil {
						return err
					}
				}
				logPath := path.Join(logdir, "heedy.log")
				f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
				if err != nil {
					return err
				}
				logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true, DisableColors: true})
				logrus.SetOutput(f)
				a.LogFile = f
			}
		}

		if a.Config.Verbose {
			logrus.SetLevel(logrus.DebugLevel) // Force debug level
			b, err := json.MarshalIndent(a.Config, "", " ")
			if err != nil {
				return err
			}
			logrus.Debug(string(b))
		}
	}

	// Validate the configuration
	return nil
}

// Abs returns config-relative absolute paths
func (a *Assets) Abs(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	// Folderpath is always absolute
	return filepath.Join(a.FolderPath, p)
}

// DataAbs returns config-relative absolute paths
func (a *Assets) DataAbs(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(a.DataDir(), p)
}

// DataDir returns the directory where data is stored
func (a *Assets) DataDir() string {
	return path.Join(a.FolderPath, "data")
}

// PluginDir returns the directory where plugin data is stored
func (a *Assets) PluginDir() string {
	return path.Join(a.FolderPath, "plugins")
}

func (a *Assets) LogDir() string {
	if a.Config == nil || a.Config.LogDir == nil || *a.Config.LogDir == "stdout" {
		return "stdout"
	}
	return a.Abs(*a.Config.LogDir)
}

func (a *Assets) AddAdmin(username string) error {
	a.Config.Lock()
	defer a.Config.Unlock()
	if a.Config.AdminUsers == nil {
		au := []string{}
		a.Config.AdminUsers = &au
	}

	// Check if the admin user already exists
	for _, v := range *a.Config.AdminUsers {
		if v == username {
			return nil
		}
	}

	// Append the user to current configuration
	au := append(*a.Config.AdminUsers, username)
	a.Config.AdminUsers = &au

	c := NewConfiguration()
	c.AdminUsers = &au

	err := WriteConfig(path.Join(a.FolderPath, "heedy.conf"), c)
	if err != nil {
		au = au[:len(au)-1]
	}

	return err
}

func (a *Assets) RemAdmin(username string) error {
	a.Config.Lock()
	defer a.Config.Unlock()
	if a.Config.AdminUsers == nil {
		return nil
	}

	// Check if the admin user already exists
	for i, v := range *a.Config.AdminUsers {
		if v == username {
			// The username exists
			au := *a.Config.AdminUsers
			au[len(au)-1], au[i] = au[i], au[len(au)-1]
			au = au[:len(au)-1]

			a.Config.AdminUsers = &au

			c := NewConfiguration()
			c.AdminUsers = &au

			err := WriteConfig(path.Join(a.FolderPath, "heedy.conf"), c)
			if err != nil {
				au = append(au, username)
			}
			return err
		}
	}
	// The username didn't exist
	return nil
}

func (a *Assets) IsAdmin(username string) bool {
	return a.Config.UserIsAdmin(username)
}

func (a *Assets) SwapAdmin(username, newname string) error {
	if a.IsAdmin(username) {
		err := a.AddAdmin(newname)

		if username != newname && err == nil {
			err = a.RemAdmin(username)
		}
		return err
	}
	return nil
}

func (a *Assets) Close() error {
	if a.LogFile != nil {
		logrus.SetOutput(os.Stdout)
		a.LogFile.Close()
	}
	return nil
}

// Open opens the assets in a given configuration path
func Open(configPath string, override *Configuration) (*Assets, error) {
	a := &Assets{
		FolderPath:     configPath,
		ConfigOverride: override,
	}

	return a, a.Reload()
}
