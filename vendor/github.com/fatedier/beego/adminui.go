// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package beego

var indexTpl = `
{{define "content"}}
<h1>Beego Admin Dashboard</h1>
<p>
For detail usage please check our document:
</p>
<p>
<a target="_blank" href="http://beego.me/docs/module/toolbox.md">Toolbox</a>
</p>
<p>
<a target="_blank" href="http://beego.me/docs/advantage/monitor.md">Live Monitor</a>
</p>
{{.Content}}
{{end}}`

var profillingTpl = `
{{define "content"}}
<h1>{{.Title}}</h1>
<pre id="content">
<div>{{.Content}}</div>
</pre>
{{end}}`

var defaultScriptsTpl = ``

var gcAjaxTpl = `
{{define "scripts"}}
<script type="text/javascript">
	var app = app || {};
(function() {
	app.$el = $('#content');
	app.getGc = function() {
		var that = this;
		$.ajax("/prof?command=gc%20summary&format=json").done(function(data) {
			that.$el.append($('<p>' + data.Content + '</p>'));
		});
	};
	$(document).ready(function() {
		setInterval(function() {
			app.getGc();
		}, 3000);
	});
})();
</script>
{{end}}
`

var qpsTpl = `{{define "content"}}
<h1>Requests statistics</h1>
<table class="table table-striped table-hover ">
	<thead>
	<tr>
	{{range .Content.Fields}}
		<th>
		{{.}}
		</th>
	{{end}}
	</tr>
	</thead>

	<tbody>
	{{range $i, $elem := .Content.Data}}

	<tr>
	    <td>{{index $elem 0}}</td>
	    <td>{{index $elem 1}}</td>
	    <td>{{index $elem 2}}</td>
	    <td data-order="{{index $elem 3}}">{{index $elem 4}}</td>
	    <td data-order="{{index $elem 5}}">{{index $elem 6}}</td>
	    <td data-order="{{index $elem 7}}">{{index $elem 8}}</td>
	    <td data-order="{{index $elem 9}}">{{index $elem 10}}</td>
	</tr>
	{{end}}
	</tbody>

</table>
{{end}}`

var configTpl = `
{{define "content"}}
<h1>Configurations</h1>
<pre>
{{range $index, $elem := .Content}}
{{$index}}={{$elem}}
{{end}}
</pre>
{{end}}
`

var routerAndFilterTpl = `{{define "content"}}


<h1>{{.Title}}</h1>

{{range .Content.Methods}}

<div class="panel panel-default">
<div class="panel-heading lead success"><strong>{{.}}</strong></div>
<div class="panel-body">
<table class="table table-striped table-hover ">
	<thead>
	<tr>
	{{range $.Content.Fields}}
		<th>
		{{.}}
		</th>
	{{end}}
	</tr>
	</thead>

	<tbody>
	{{$slice := index $.Content.Data .}}
	{{range $i, $elem := $slice}}

	<tr>
		{{range $elem}}
			<td>
			{{.}}
			</td>
		{{end}}
	</tr>

	{{end}}
	</tbody>

</table>
</div>
</div>
{{end}}


{{end}}`

var tasksTpl = `{{define "content"}}

<h1>{{.Title}}</h1>

{{if .Message }}
{{ $messageType := index .Message 0}}
<p class="message
{{if eq "error" $messageType}}
bg-danger
{{else if eq "success" $messageType}}
bg-success
{{else}}
bg-warning
{{end}}
">
{{index .Message 1}}
</p>
{{end}}


<table class="table table-striped table-hover ">
<thead>
<tr>
{{range .Content.Fields}}
<th>
{{.}}
</th>
{{end}}
</tr>
</thead>

<tbody>
{{range $i, $slice := .Content.Data}}
<tr>
	{{range $slice}}
	<td>
	{{.}}
	</td>
	{{end}}
	<td>
	<a class="btn btn-primary btn-sm" href="/task?taskname={{index $slice 0}}">Run</a>
	</td>
</tr>
{{end}}
</tbody>
</table>

{{end}}`

var healthCheckTpl = `
{{define "content"}}

<h1>{{.Title}}</h1>
<table class="table table-striped table-hover ">
<thead>
<tr>
{{range .Content.Fields}}
	<th>
	{{.}}
	</th>
{{end}}
</tr>
</thead>
<tbody>
{{range $i, $slice := .Content.Data}}
	{{ $header := index $slice 0}}
	{{ if eq "success" $header}}
	<tr class="success">
	{{else if eq "error" $header}}
	<tr class="danger">
	{{else}}
	<tr>
	{{end}}
		{{range $j, $elem := $slice}}
		{{if ne $j 0}}
		<td>
		{{$elem}}
		</td>
		{{end}}
		{{end}}
		<td>
		{{$header}}
		</td>
	</tr>
{{end}}

</tbody>
</table>
{{end}}`

// The base dashboardTpl
var dashboardTpl = `
<!DOCTYPE html>
<html lang="en">
<head>
<!-- Meta, title, CSS, favicons, etc. -->
<meta charset="utf-8">
<meta http-equiv="X-UA-Compatible" content="IE=edge">
<meta name="viewport" content="width=device-width, initial-scale=1">

<title>

Welcome to Beego Admin Dashboard

</title>

<link href="//maxcdn.bootstrapcdn.com/bootstrap/3.2.0/css/bootstrap.min.css" rel="stylesheet">
<link href="//cdn.datatables.net/plug-ins/725b2a2115b/integration/bootstrap/3/dataTables.bootstrap.css" rel="stylesheet">

<style type="text/css">
ul.nav li.dropdown:hover > ul.dropdown-menu {
	display: block;    
}
#logo {
	width: 102px;
	height: 32px;
	margin-top: 5px;
}
.message {
	padding: 15px;
}
</style>

</head>
<body>

<header class="navbar navbar-default navbar-static-top bs-docs-nav" id="top" role="banner">
<div class="container">
<div class="navbar-header">
<button class="navbar-toggle" type="button" data-toggle="collapse" data-target=".bs-navbar-collapse">
<span class="sr-only">Toggle navigation</span>
<span class="icon-bar"></span>
<span class="icon-bar"></span>
<span class="icon-bar"></span>
</button>

<a href="/">
<img id="logo" src="data:image/png;base64,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"/>
</a>

</div>
<nav class="collapse navbar-collapse bs-navbar-collapse" role="navigation">
<ul class="nav navbar-nav">
<li>
<a href="/qps">
Requests statistics
</a>
</li>
<li>

<li class="dropdown">
<a href="#" class="dropdown-toggle disabled" data-toggle="dropdown">Performance profiling<span class="caret"></span></a>
<ul class="dropdown-menu" role="menu">

<li><a href="/prof?command=lookup goroutine">lookup goroutine</a></li>
<li><a href="/prof?command=lookup heap">lookup heap</a></li>
<li><a href="/prof?command=lookup threadcreate">lookup threadcreate</a></li>
<li><a href="/prof?command=lookup block">lookup block</a></li>
<li><a href="/prof?command=get cpuprof">get cpuprof</a></li>
<li><a href="/prof?command=get memprof">get memprof</a></li>
<li><a href="/prof?command=gc summary">gc summary</a></li>

</ul>
</li>

<li>
<a href="/healthcheck">
Healthcheck
</a>
</li>

<li>
<a href="/task" class="dropdown-toggle disabled" data-toggle="dropdown">Tasks</a>
</li>

<li class="dropdown">
<a href="#" class="dropdown-toggle disabled" data-toggle="dropdown">Config Status<span class="caret"></span></a>
<ul class="dropdown-menu" role="menu">
<li><a href="/listconf?command=conf">Configs</a></li>
<li><a href="/listconf?command=router">Routers</a></li>
<li><a href="/listconf?command=filter">Filters</a></li>
</ul>
</li>
</ul>
</nav>
</div>
</header>

<div class="container">
{{template "content" .}}
</div>

<script src="//code.jquery.com/jquery-1.11.1.min.js"></script>
<script src="//maxcdn.bootstrapcdn.com/bootstrap/3.2.0/js/bootstrap.min.js"></script>
<script src="//cdn.datatables.net/1.10.2/js/jquery.dataTables.min.js"></script>
<script src="//cdn.datatables.net/plug-ins/725b2a2115b/integration/bootstrap/3/dataTables.bootstrap.js
"></script>

<script type="text/javascript">
$(document).ready(function() {
    $('.table').dataTable();
});
</script>
{{template "scripts" .}}
</body>
</html>
`
