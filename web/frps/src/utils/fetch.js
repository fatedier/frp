import { Message } from 'element-ui'

export default function(api, init = {}) {
  return new Promise((resolve, reject) => {
    fetch(`/api/${api}`, Object.assign({ credentials: 'include' }, init))
      .then(res => {
        resolve(res)
      })
      .catch(err => {
        Message.error(err.message)
        reject()
      })
  })
}
