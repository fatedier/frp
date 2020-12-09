import fetch from '@/utils/fetch'
import { Message } from 'element-ui'

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
      Message.warning('Get server info from frps failed!')
      commit('SET_SERVER_INFO', null)
      return
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
