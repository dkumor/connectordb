/**
 * This file contains the main components that are used in Heedy's core.
 * The components here will be stable over minor versions, so plugins
 * are free to use them for their own UI
 */

import Loading from "./components/loading.vue";
import Icon from "./components/icon.vue";
import IconEditor from "./components/icon_editor.vue";
import ScopeEditor from "./components/scope_editor.vue";
import TagEditor from "./components/tag_editor.vue";
import PageContainer from "./components/page_container.vue";
import CardPage from "./components/card_page.vue";
import NotFound from "./components/404.vue";
import Header from "./components/header.vue";
import ObjectHeader from "./components/object_header.vue";
import ObjectUpdater from "./components/object_updater.vue";
import ObjectCreator from "./components/object_creator.vue";
import ObjectPicker from "./components/object_picker.vue";
import AppButton from "./components/app_button.vue";
import MenuToolbarItems from "./components/menu_toolbar_items.vue";
import JSF from "./components/jsf.vue";
import MD from "./components/md.vue";

function register(Vue) {
  Vue.component("h-loading", Loading);
  Vue.component("h-icon", Icon);
  Vue.component("h-icon-editor", IconEditor);
  Vue.component("h-scope-editor", ScopeEditor);
  Vue.component("h-tag-editor", TagEditor);
  Vue.component("h-page-container", PageContainer);
  Vue.component("h-card-page", CardPage);
  Vue.component("h-not-found", NotFound);
  Vue.component("h-header", Header);
  Vue.component("h-object-header", ObjectHeader);
  Vue.component("h-object-updater", ObjectUpdater);
  Vue.component("h-object-picker", ObjectPicker);
  Vue.component("h-app-button", AppButton);
  Vue.component("h-object-creator", ObjectCreator);
  Vue.component("h-menu-toolbar-items", MenuToolbarItems);
  Vue.component("h-jsf", JSF);
  Vue.component("h-md", MD);
}

export {
  Loading,
  Icon,
  IconEditor,
  ScopeEditor,
  TagEditor,
  PageContainer,
  CardPage,
  NotFound,
  Header,
  ObjectHeader,
  ObjectUpdater,
  ObjectCreator,
  ObjectPicker,
  AppButton,
  MenuToolbarItems,
  JSF, MD
};

export default register;
