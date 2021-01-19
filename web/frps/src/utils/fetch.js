import { Message } from 'element-ui'

export default function(api, init = {}) {
  return new Promise(resolve => {
    fetch(`/api/${api}`, Object.assign({ credentials: 'include' }, init))
      .then(res => {
        if (res.status < 200 || res.status >= 300) {
          Message.warning('Get server info from frps failed!')
          resolve()
          return
        }

        resolve(res ? res.json() : undefined)
      })
      .catch(err => {
        this.$message.error(err.message)
        resolve()
      })
  })
}
