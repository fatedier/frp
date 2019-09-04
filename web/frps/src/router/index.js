
var uriInfo = new URL(window.location.href); 
var str = uriInfo.pathname;  
var hasStatic = str.lastIndexOf("static"); 

if (hasStatic > 0){ 
    window.base_path = str.substring(0,  hasStatic);  
}else {
    window.base_path = "/"; 
}

// console.log("windows base url is "+ window.base_path); 
 
import Vue from 'vue'
import Router from 'vue-router'
import Overview from '../components/Overview.vue'
import ProxiesTcp from '../components/ProxiesTcp.vue'
import ProxiesUdp from '../components/ProxiesUdp.vue'
import ProxiesHttp from '../components/ProxiesHttp.vue'
import ProxiesHttps from '../components/ProxiesHttps.vue'
import ProxiesStcp from '../components/ProxiesStcp.vue'
 
Vue.use(Router)

export default new Router({ 
    routes: [{
        path: '/',
        name: 'Overview',
        component: Overview
    }, {
        path: '/proxies/tcp',
        name: 'ProxiesTcp',
        component: ProxiesTcp
    }, {
        path: '/proxies/udp',
        name: 'ProxiesUdp',
        component: ProxiesUdp
    }, {
        path: '/proxies/http',
        name: 'ProxiesHttp',
        component: ProxiesHttp
    }, {
        path: '/proxies/https',
        name: 'ProxiesHttps',
        component: ProxiesHttps
    }, {
        path: '/proxies/stcp',
        name: 'ProxiesStcp',
        component: ProxiesStcp
    }]
})
