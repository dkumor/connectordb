import Vue from "../../dist/vue.mjs";
// import api from "../../util.mjs";

export default {
  state: {
    visualizationTypes: {},
    types: {}
  },
  mutations: {
    addTSVisualizationType(state, v) {
      Vue.set(state.visualizationTypes, v.key, v.component);
    },
    addTSType(state, v) {
      Vue.set(state.types, v.key, v);
    }
  },
};
