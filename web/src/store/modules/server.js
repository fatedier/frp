import Vue from 'vue'
import Vuex from 'vuex'
import fetch from '@/utils/fetch'
Vue.use(Vuex)

const state = {
  serverInfo: null
}

const mutations = {
  SET_SERVER_INFO(state, serverInfo) {
    state.serverInfo = serverInfo
  }
}

const actions = {
  async fetchServerInfo({ commit }) {
    const res = await fetch('serverinfo')
    if (!res.ok) {
      this.$message.warning('Get server info from frps failed!')
      commit('SET_SERVER_INFO', null)
    }

    commit('SET_SERVER_INFO', (await res.json()) || null)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
