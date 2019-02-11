<template>
    <div id="app">
        <header class="grid-content header-color">
            <el-row>
                <a class="brand" href="#">frp client</a>
            </el-row>
        </header>
        <section>
            <el-row :gutter="20">
                <el-col id="side-nav" :xs="24" :md="4">
                    <el-menu default-active="1" mode="vertical" theme="light" router="false" @select="handleSelect">
                        <el-menu-item index="/">Overview</el-menu-item>
                        <el-menu-item index="/configure">Configure</el-menu-item>
                        <el-menu-item index="">Help</el-menu-item>
                    </el-menu>
				</el-col>

				<el-col :xs="24" :md="20">
                    <div id="content">
                    <router-view></router-view>
                    </div>
				</el-col>
		</el-row>
	</section>
	<footer></footer>
</div>
</template>

<script>
    export default {
        methods: {
            handleSelect(key, path) {
                if (key == '') {
                    window.open("https://github.com/fatedier/frp")
                }
            }
        }
    }
</script>

<style>
    body {
        background-color: #fafafa;
        margin: 0px;
        font-family: -apple-system,BlinkMacSystemFont,Helvetica Neue,sans-serif;
    }
    
    header {
        width: 100%;
        height: 60px;
    }
    
    .header-color {
        background: #58B7FF;
    }
    
    #content {
        margin-top: 20px;
        padding-right: 40px;
    }
    
    .brand {
        color: #fff;
        background-color: transparent;
        margin-left: 20px;
        float: left;
        line-height: 25px;
        font-size: 25px;
        padding: 15px 15px;
        height: 30px;
        text-decoration: none;
    }
</style>
