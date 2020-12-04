<template>
  <div>
    <el-row>
      <el-col :md="24">
        <div>
          <el-table :data="status" stripe style="width: 100%" :default-sort="{ prop: 'type', order: 'ascending' }">
            <el-table-column prop="name" label="name" />
            <el-table-column prop="type" label="type" width="150" />
            <el-table-column prop="local_addr" label="local address" width="200" />
            <el-table-column prop="plugin" label="plugin" width="200" />
            <el-table-column prop="remote_addr" label="remote address" />
            <el-table-column prop="status" label="status" width="150" />
            <el-table-column prop="err" label="info" />
          </el-table>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script>
export default {
  data() {
    return {
      status: []
    }
  },
  created() {
    this.fetchData()
  },
  methods: {
    async fetchData() {
      const res = await this.$fetch('status')
      if (!res.ok) {
        this.$message({
          showClose: true,
          message: 'Get status info from frpc failed!',
          type: 'warning'
        })
        return
      }

      const json = await res.json()
      this.status = []
      for (const s of json.tcp) {
        this.status.push(s)
      }
      for (const s of json.udp) {
        this.status.push(s)
      }
      for (const s of json.http) {
        this.status.push(s)
      }
      for (const s of json.https) {
        this.status.push(s)
      }
      for (const s of json.stcp) {
        this.status.push(s)
      }
      for (const s of json.xtcp) {
        this.status.push(s)
      }
    }
  }
}
</script>

<style>
</style>
