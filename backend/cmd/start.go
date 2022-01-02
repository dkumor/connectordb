package cmd

import (
	"github.com/heedy/heedy/backend/updater"

	"github.com/spf13/cobra"
)

var StartCmd = &cobra.Command{
	Use:   "start [location of database]",
	Short: "Starts heedy in the background",
	Long:  `Starts a background heedy process using the passed database. If not folder is specified, uses the default database location.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		directory, err := GetDirectory(args)
		if err != nil {
			return err
		}

		return updater.StartHeedy(directory, false)
	},
}

func init() {
	RootCmd.AddCommand(StartCmd)
}
