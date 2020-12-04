<template>
  <div>
    <el-row id="head">
      <el-button type="primary" @click="fetchData">Refresh</el-button>
      <el-button type="primary" @click="uploadConfig">Upload</el-button>
    </el-row>
    <el-input v-model="textarea" type="textarea" autosize placeholder="frpc configrue file, can not be empty..." />
  </div>
</template>

<script>
export default {
  data() {
    return {
      textarea: ''
    }
  },
  watch: {
    $route: 'fetchData'
  },
  created() {
    this.fetchData()
  },
  methods: {
    fetchData() {
      fetch('/api/config', { credentials: 'include' })
        .then(res => {
          return res.text()
        })
        .then(text => {
          this.textarea = text
        })
        .catch(err => {
          this.$message({
            showClose: true,
            message: 'Get configure content from frpc failed!',
            type: 'warning'
          })
          return err
        })
    },
    uploadConfig() {
      this.$confirm('This operation will upload your frpc configure file content and hot reload it, do you want to continue?', 'Notice', {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning'
      })
        .then(() => {
          if (!this.textarea) {
            this.$message({
              type: 'warning',
              message: 'Configure content can not be empty!'
            })
            return
          }

          fetch('/api/config', {
            credentials: 'include',
            method: 'PUT',
            body: this.textarea
          })
            .then(() => {
              fetch('/api/reload', { credentials: 'include' })
                .then(() => {
                  this.$message({
                    type: 'success',
                    message: 'Success'
                  })
                })
                .catch(err => {
                  this.$message({
                    showClose: true,
                    message: 'Reload frpc configure file error, ' + err,
                    type: 'warning'
                  })
                })
            })
            .catch(err => {
              this.$message({
                showClose: true,
                message: 'Put config to frpc and hot reload failed!',
                type: 'warning'
              })
              return err
            })
        })
        .catch(() => {
          this.$message({
            type: 'info',
            message: 'Canceled'
          })
        })
    }
  }
}
</script>

<style>
#head {
  margin-bottom: 30px;
}
</style>
