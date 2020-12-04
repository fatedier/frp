<template>
  <div class="configure-container">
    <el-row id="head">
      <el-button type="primary" @click="refresh">Refresh</el-button>
      <el-button type="primary" @click="uploadConfig">Upload</el-button>
    </el-row>
    <el-input v-model="textarea" type="textarea" autosize placeholder="frpc configrue file, can not be empty..." :spellcheck="false" />
  </div>
</template>

<script>
export default {
  data() {
    return {
      textarea: ''
    }
  },
  created() {
    this.fetchData()
  },
  methods: {
    async fetchData() {
      const res = await this.$fetch('config')
      if (!res.ok) {
        this.$message({
          showClose: true,
          message: 'Get configure content from frpc failed!',
          type: 'warning'
        })
        return
      }

      this.textarea = await res.text()
      return this.textarea
    },
    async refresh() {
      if (await this.fetchData()) {
        this.$message.success('Success')
      }
    },
    uploadConfig() {
      this.$confirm('This operation will upload your frpc configure file content and hot reload it, do you want to continue?', 'Notice', {
        confirmButtonText: 'Yes',
        cancelButtonText: 'No',
        type: 'warning'
      }).then(async () => {
        if (!this.textarea) {
          this.$message({
            type: 'warning',
            message: 'Configure content can not be empty!'
          })
          return
        }

        const putRes = await this.$fetch('config', {
          method: 'PUT',
          body: this.textarea
        })
        if (!putRes.ok) {
          this.$message({
            showClose: true,
            message: 'Put config to frpc and hot reload failed!',
            type: 'warning'
          })
          return
        }

        const reloadRes = await this.$fetch('reload')
        if (!reloadRes.ok) {
          this.$message({
            showClose: true,
            message: 'Reload frpc configure file error, ' + reloadRes.statusText,
            type: 'warning'
          })
          return
        }

        this.$message({
          type: 'success',
          message: 'Success'
        })
      })
    }
  }
}
</script>

<style>
.configure-container {
  margin: 20px;
}
#head {
  margin-bottom: 20px;
}
</style>
