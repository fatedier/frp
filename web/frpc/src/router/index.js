import Vue from 'vue'
import Router from 'vue-router'
import Overview from '../components/Overview.vue'
import Configure from '../components/Configure.vue'

Vue.use(Router)

export default new Router({
    routes: [{
        path: '/',
        name: 'Overview',
        component: Overview
    },{
        path: '/configure',
        name: 'Configure',
        component: Configure,
    }]
})
