const path = require('path')
const appType = process.env.VUE_APP_TYPE

function resolve(dir) {
  return path.join(__dirname, dir)
}

function getProxyTargetPort() {
  switch (appType) {
    case 'frps':
      return 8081
    case 'frpc':
      return 8082
    default:
      return 8080
  }
}

module.exports = {
  publicPath: './',
  outputDir: `./dist/${appType}`,
  productionSourceMap: false,
  devServer: {
    host: '127.0.0.1',
    port: 8010,
    proxy: {
      '/api/': {
        target: `http://127.0.0.1:${getProxyTargetPort()}/api`,
        changeOrigin: true,
        pathRewrite: {
          '^/api': ''
        }
      }
    }
  },
  chainWebpack(config) {
    config.plugins.delete('preload')
    config.plugins.delete('prefetch')

    // set svg-sprite-loader
    config.module
      .rule('svg')
      .exclude.add(resolve('src/icons'))
      .end()
    config.module
      .rule('icons')
      .test(/\.svg$/)
      .include.add(resolve('src/icons'))
      .end()
      .use('svg-sprite-loader')
      .loader('svg-sprite-loader')
      .options({
        symbolId: 'icon-[name]'
      })
      .end()

    config.when(process.env.NODE_ENV === 'development', config => config.devtool('eval-source-map'))
  }
}
