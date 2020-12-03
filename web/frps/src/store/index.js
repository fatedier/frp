import Vue from 'vue'
import Vuex from 'vuex'
import fetch from '@/utils/fetch'
Vue.use(Vuex)

const store = new Vuex.Store({
  state: {
    serverInfo: null
  },
  mutations: {
    SET_SERVER_INFO(state, serverInfo) {
      state.serverInfo = serverInfo
    }
  },
  actions: {
    async fetchServerInfo({ commit }) {
      const json = await fetch('serverinfo')
      commit('SET_SERVER_INFO', json || null)
      return json
    }
  }
})

export default store
