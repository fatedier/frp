/*
Package pagination provides utilities to setup a paginator within the
context of a http request.

Usage

In your beego.Controller:

 package controllers

 import "github.com/astaxie/beego/utils/pagination"

 type PostsController struct {
   beego.Controller
 }

 func (this *PostsController) ListAllPosts() {
     // sets this.Data["paginator"] with the current offset (from the url query param)
     postsPerPage := 20
     paginator := pagination.SetPaginator(this.Ctx, postsPerPage, CountPosts())

     // fetch the next 20 posts
     this.Data["posts"] = ListPostsByOffsetAndLimit(paginator.Offset(), postsPerPage)
 }


In your view templates:

 {{if .paginator.HasPages}}
 <ul class="pagination pagination">
     {{if .paginator.HasPrev}}
         <li><a href="{{.paginator.PageLinkFirst}}">{{ i18n .Lang "paginator.first_page"}}</a></li>
         <li><a href="{{.paginator.PageLinkPrev}}">&laquo;</a></li>
     {{else}}
         <li class="disabled"><a>{{ i18n .Lang "paginator.first_page"}}</a></li>
         <li class="disabled"><a>&laquo;</a></li>
     {{end}}
     {{range $index, $page := .paginator.Pages}}
         <li{{if $.paginator.IsActive .}} class="active"{{end}}>
             <a href="{{$.paginator.PageLink $page}}">{{$page}}</a>
         </li>
     {{end}}
     {{if .paginator.HasNext}}
         <li><a href="{{.paginator.PageLinkNext}}">&raquo;</a></li>
         <li><a href="{{.paginator.PageLinkLast}}">{{ i18n .Lang "paginator.last_page"}}</a></li>
     {{else}}
         <li class="disabled"><a>&raquo;</a></li>
         <li class="disabled"><a>{{ i18n .Lang "paginator.last_page"}}</a></li>
     {{end}}
 </ul>
 {{end}}

See also

http://beego.me/docs/mvc/view/page.md

*/
package pagination
