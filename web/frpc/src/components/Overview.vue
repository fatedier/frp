<template>
    <div>
        <el-row>
            <el-col :md="24">
                <div>
                    <el-table :data="status" stripe style="width: 100%" :default-sort="{prop: 'type', order: 'ascending'}">
                        <el-table-column prop="name" label="name"></el-table-column>
                        <el-table-column prop="type" label="type" width="150"></el-table-column>
                        <el-table-column prop="local_addr" label="local address" width="200"></el-table-column>
                        <el-table-column prop="plugin" label="plugin" width="200"></el-table-column>
                        <el-table-column prop="remote_addr" label="remote address"></el-table-column>
                        <el-table-column prop="status" label="status" width="150"></el-table-column>
                        <el-table-column prop="err" label="info"></el-table-column>
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
                status: new Array(),
            }
        },
        created() {
            this.fetchData()
        },
        watch: {
            '$route': 'fetchData'
        },
        methods: {
            fetchData() {
                fetch('/api/status', {credentials: 'include'})
              .then(res => {
                return res.json()
              }).then(json => {
                this.status = new Array()
                for (let s of json.tcp) {
                    this.status.push(s)
                }
                for (let s of json.udp) {
                    this.status.push(s)
                }
                for (let s of json.http) {
                    this.status.push(s)
                }
                for (let s of json.https) {
                    this.status.push(s)
                }
                for (let s of json.stcp) {
                    this.status.push(s)
                }
                for (let s of json.sudp) {
                    this.status.push(s)
                }
                for (let s of json.xtcp) {
                    this.status.push(s)
                }
              }).catch( err => {
                  this.$message({
                      showClose: true,
                      message: 'Get status info from frpc failed!',
                      type: 'warning'
                    })
              })
            }
        }
    }
</script>

<style>
</style>
