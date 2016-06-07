
#Gin Web Framework
<img align="right" src="https://raw.githubusercontent.com/gin-gonic/gin/master/logo.jpg">
[![Build Status](https://travis-ci.org/gin-gonic/gin.svg)](https://travis-ci.org/gin-gonic/gin)
[![Coverage Status](https://coveralls.io/repos/gin-gonic/gin/badge.svg?branch=master)](https://coveralls.io/r/gin-gonic/gin?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/gin-gonic/gin)](https://goreportcard.com/report/github.com/gin-gonic/gin)
[![GoDoc](https://godoc.org/github.com/gin-gonic/gin?status.svg)](https://godoc.org/github.com/gin-gonic/gin)
[![Join the chat at https://gitter.im/gin-gonic/gin](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/gin-gonic/gin?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

Gin is a web framework written in Go (Golang). It features a martini-like API with much better performance, up to 40 times faster thanks to [httprouter](https://github.com/julienschmidt/httprouter). If you need performance and good productivity, you will love Gin.



![Gin console logger](https://gin-gonic.github.io/gin/other/console.png)

```sh
$ cat test.go
```
```go
package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run() // listen and server on 0.0.0.0:8080
}
```

## Benchmarks

Gin uses a custom version of [HttpRouter](https://github.com/julienschmidt/httprouter)  

[See all benchmarks](/BENCHMARKS.md)


Benchmark name 					| (1) 		| (2) 		| (3) 		| (4)
--------------------------------|----------:|----------:|----------:|------:
BenchmarkAce_GithubAll 			| 10000 	| 109482 	| 13792 	| 167
BenchmarkBear_GithubAll 		| 10000 	| 287490 	| 79952 	| 943
BenchmarkBeego_GithubAll 		| 3000 		| 562184 	| 146272 	| 2092
BenchmarkBone_GithubAll 		| 500 		| 2578716 	| 648016 	| 8119
BenchmarkDenco_GithubAll 		| 20000 	| 94955 	| 20224 	| 167
BenchmarkEcho_GithubAll 		| 30000 	| 58705 	| 0 		| 0
**BenchmarkGin_GithubAll** 		| **30000** | **50991** | **0** 	| **0**
BenchmarkGocraftWeb_GithubAll 	| 5000 		| 449648 	| 133280 	| 1889
BenchmarkGoji_GithubAll 		| 2000 		| 689748 	| 56113 	| 334
BenchmarkGoJsonRest_GithubAll 	| 5000 		| 537769 	| 135995 	| 2940
BenchmarkGoRestful_GithubAll 	| 100 		| 18410628 	| 797236 	| 7725
BenchmarkGorillaMux_GithubAll 	| 200 		| 8036360 	| 153137 	| 1791
BenchmarkHttpRouter_GithubAll 	| 20000 	| 63506 	| 13792 	| 167
BenchmarkHttpTreeMux_GithubAll 	| 10000 	| 165927 	| 56112 	| 334
BenchmarkKocha_GithubAll 		| 10000 	| 171362 	| 23304 	| 843
BenchmarkMacaron_GithubAll 		| 2000 		| 817008 	| 224960 	| 2315
BenchmarkMartini_GithubAll 		| 100 		| 12609209 	| 237952 	| 2686
BenchmarkPat_GithubAll 			| 300 		| 4830398 	| 1504101 	| 32222
BenchmarkPossum_GithubAll 		| 10000 	| 301716 	| 97440 	| 812
BenchmarkR2router_GithubAll 	| 10000 	| 270691 	| 77328 	| 1182
BenchmarkRevel_GithubAll 		| 1000 		| 1491919 	| 345553 	| 5918
BenchmarkRivet_GithubAll 		| 10000 	| 283860 	| 84272 	| 1079
BenchmarkTango_GithubAll 		| 5000 		| 473821 	| 87078 	| 2470
BenchmarkTigerTonic_GithubAll 	| 2000 		| 1120131 	| 241088 	| 6052
BenchmarkTraffic_GithubAll 		| 200 		| 8708979 	| 2664762 	| 22390
BenchmarkVulcan_GithubAll 		| 5000 		| 353392 	| 19894 	| 609
BenchmarkZeus_GithubAll 		| 2000 		| 944234 	| 300688 	| 2648

(1): Total Repetitions  
(2): Single Repetition Duration (ns/op)  
(3): Heap Memory (B/op)  
(4): Average Allocations per Repetition (allocs/op)  

##Gin v1. stable

- [x] Zero allocation router.
- [x] Still the fastest http router and framework. From routing to writing.
- [x] Complete suite of unit tests
- [x] Battle tested
- [x] API frozen, new releases will not break your code.


## Start using it
1. Download and install it:

    ```sh
    $ go get github.com/gin-gonic/gin
    ```

2. Import it in your code:

    ```go
    import "github.com/gin-gonic/gin"
    ```

3. (Optional) Import `net/http`. This is required for example if using constants such as `http.StatusOK`.

    ```go
    import "net/http"
    ```

##API Examples

#### Using GET, POST, PUT, PATCH, DELETE and OPTIONS

```go
func main() {
	// Creates a gin router with default middleware:
	// logger and recovery (crash-free) middleware
	router := gin.Default()

	router.GET("/someGet", getting)
	router.POST("/somePost", posting)
	router.PUT("/somePut", putting)
	router.DELETE("/someDelete", deleting)
	router.PATCH("/somePatch", patching)
	router.HEAD("/someHead", head)
	router.OPTIONS("/someOptions", options)

	// By default it serves on :8080 unless a
	// PORT environment variable was defined.
	router.Run()
	// router.Run(":3000") for a hard coded port
}
```

#### Parameters in path

```go
func main() {
	router := gin.Default()

	// This handler will match /user/john but will not match neither /user/ or /user
	router.GET("/user/:name", func(c *gin.Context) {
		name := c.Param("name")
		c.String(http.StatusOK, "Hello %s", name)
	})

	// However, this one will match /user/john/ and also /user/john/send
	// If no other routers match /user/john, it will redirect to /user/john/
	router.GET("/user/:name/*action", func(c *gin.Context) {
		name := c.Param("name")
		action := c.Param("action")
		message := name + " is " + action
		c.String(http.StatusOK, message)
	})

	router.Run(":8080")
}
```

#### Querystring parameters
```go
func main() {
	router := gin.Default()

	// Query string parameters are parsed using the existing underlying request object.
	// The request responds to a url matching:  /welcome?firstname=Jane&lastname=Doe
	router.GET("/welcome", func(c *gin.Context) {
		firstname := c.DefaultQuery("firstname", "Guest")
		lastname := c.Query("lastname") // shortcut for c.Request.URL.Query().Get("lastname")

		c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
	})
	router.Run(":8080")
}
```

### Multipart/Urlencoded Form

```go
func main() {
	router := gin.Default()

	router.POST("/form_post", func(c *gin.Context) {
		message := c.PostForm("message")
		nick := c.DefaultPostForm("nick", "anonymous")

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
			"nick":    nick,
		})
	})
	router.Run(":8080")
}
```

### Another example: query + post form

```
POST /post?id=1234&page=1 HTTP/1.1
Content-Type: application/x-www-form-urlencoded

name=manu&message=this_is_great
```

```go
func main() {
	router := gin.Default()

	router.POST("/post", func(c *gin.Context) {

		id := c.Query("id")
		page := c.DefaultQuery("page", "0")
		name := c.PostForm("name")
		message := c.PostForm("message")

		fmt.Printf("id: %s; page: %s; name: %s; message: %s", id, page, name, message)
	})
	router.Run(":8080")
}
```

```
id: 1234; page: 1; name: manu; message: this_is_great
```

### Another example: upload file

References issue [#548](https://github.com/gin-gonic/gin/issues/548).

```go
func main() {
	router := gin.Default()

	router.POST("/upload", func(c *gin.Context) {

	        file, header , err := c.Request.FormFile("upload")
	        filename := header.Filename
	        fmt.Println(header.Filename)
	        out, err := os.Create("./tmp/"+filename+".png")
	        if err != nil {
	            log.Fatal(err)
	        }
	        defer out.Close()
	        _, err = io.Copy(out, file)
	        if err != nil {
	            log.Fatal(err)
	        }   
	})
	router.Run(":8080")
}
```

#### Grouping routes
```go
func main() {
	router := gin.Default()

	// Simple group: v1
	v1 := router.Group("/v1")
	{
		v1.POST("/login", loginEndpoint)
		v1.POST("/submit", submitEndpoint)
		v1.POST("/read", readEndpoint)
	}

	// Simple group: v2
	v2 := router.Group("/v2")
	{
		v2.POST("/login", loginEndpoint)
		v2.POST("/submit", submitEndpoint)
		v2.POST("/read", readEndpoint)
	}

	router.Run(":8080")
}
```


#### Blank Gin without middleware by default

Use

```go
r := gin.New()
```
instead of

```go
r := gin.Default()
```


#### Using middleware
```go
func main() {
	// Creates a router without any middleware by default
	r := gin.New()

	// Global middleware
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// Per route middleware, you can add as many as you desire.
	r.GET("/benchmark", MyBenchLogger(), benchEndpoint)

	// Authorization group
	// authorized := r.Group("/", AuthRequired())
	// exactly the same as:
	authorized := r.Group("/")
	// per group middleware! in this case we use the custom created
	// AuthRequired() middleware just in the "authorized" group.
	authorized.Use(AuthRequired())
	{
		authorized.POST("/login", loginEndpoint)
		authorized.POST("/submit", submitEndpoint)
		authorized.POST("/read", readEndpoint)

		// nested group
		testing := authorized.Group("testing")
		testing.GET("/analytics", analyticsEndpoint)
	}

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
```

#### Model binding and validation

To bind a request body into a type, use model binding. We currently support binding of JSON, XML and standard form values (foo=bar&boo=baz).

Note that you need to set the corresponding binding tag on all fields you want to bind. For example, when binding from JSON, set `json:"fieldname"`.

When using the Bind-method, Gin tries to infer the binder depending on the Content-Type header. If you are sure what you are binding, you can use BindWith.

You can also specify that specific fields are required. If a field is decorated with `binding:"required"` and has a empty value when binding, the current request will fail with an error.

```go
// Binding from JSON
type Login struct {
	User     string `form:"user" json:"user" binding:"required"`
	Password string `form:"password" json:"password" binding:"required"`
}

func main() {
	router := gin.Default()

	// Example for binding JSON ({"user": "manu", "password": "123"})
	router.POST("/loginJSON", func(c *gin.Context) {
		var json Login
		if c.BindJSON(&json) == nil {
			if json.User == "manu" && json.Password == "123" {
				c.JSON(http.StatusOK, gin.H{"status": "you are logged in"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			}
		}
	})

	// Example for binding a HTML form (user=manu&password=123)
	router.POST("/loginForm", func(c *gin.Context) {
		var form Login
		// This will infer what binder to use depending on the content-type header.
		if c.Bind(&form) == nil {
			if form.User == "manu" && form.Password == "123" {
				c.JSON(http.StatusOK, gin.H{"status": "you are logged in"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"status": "unauthorized"})
			}
		}
	})

	// Listen and server on 0.0.0.0:8080
	router.Run(":8080")
}
```


###Multipart/Urlencoded binding
```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type LoginForm struct {
	User     string `form:"user" binding:"required"`
	Password string `form:"password" binding:"required"`
}

func main() {
	router := gin.Default()
	router.POST("/login", func(c *gin.Context) {
		// you can bind multipart form with explicit binding declaration:
		// c.BindWith(&form, binding.Form)
		// or you can simply use autobinding with Bind method:
		var form LoginForm
		// in this case proper binding will be automatically selected
		if c.Bind(&form) == nil {
			if form.User == "user" && form.Password == "password" {
				c.JSON(200, gin.H{"status": "you are logged in"})
			} else {
				c.JSON(401, gin.H{"status": "unauthorized"})
			}
		}
	})
	router.Run(":8080")
}
```

Test it with:
```sh
$ curl -v --form user=user --form password=password http://localhost:8080/login
```


#### XML and JSON rendering

```go
func main() {
	r := gin.Default()

	// gin.H is a shortcut for map[string]interface{}
	r.GET("/someJSON", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "hey", "status": http.StatusOK})
	})

	r.GET("/moreJSON", func(c *gin.Context) {
		// You also can use a struct
		var msg struct {
			Name    string `json:"user"`
			Message string
			Number  int
		}
		msg.Name = "Lena"
		msg.Message = "hey"
		msg.Number = 123
		// Note that msg.Name becomes "user" in the JSON
		// Will output  :   {"user": "Lena", "Message": "hey", "Number": 123}
		c.JSON(http.StatusOK, msg)
	})

	r.GET("/someXML", func(c *gin.Context) {
		c.XML(http.StatusOK, gin.H{"message": "hey", "status": http.StatusOK})
	})

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
```

####Serving static files

```go
func main() {
	router := gin.Default()
	router.Static("/assets", "./assets")
	router.StaticFS("/more_static", http.Dir("my_file_system"))
	router.StaticFile("/favicon.ico", "./resources/favicon.ico")

	// Listen and server on 0.0.0.0:8080
	router.Run(":8080")
}
```

####HTML rendering

Using LoadHTMLTemplates()

```go
func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	//router.LoadHTMLFiles("templates/template1.html", "templates/template2.html")
	router.GET("/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.tmpl", gin.H{
			"title": "Main website",
		})
	})
	router.Run(":8080")
}
```
templates/index.tmpl
```html
<html>
	<h1>
		{{ .title }}
	</h1>
</html>
```

Using templates with same name in different directories

```go
func main() {
	router := gin.Default()
	router.LoadHTMLGlob("templates/**/*")
	router.GET("/posts/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "posts/index.tmpl", gin.H{
			"title": "Posts",
		})
	})
	router.GET("/users/index", func(c *gin.Context) {
		c.HTML(http.StatusOK, "users/index.tmpl", gin.H{
			"title": "Users",
		})
	})
	router.Run(":8080")
}
```
templates/posts/index.tmpl
```html
{{ define "posts/index.tmpl" }}
<html><h1>
	{{ .title }}
</h1>
<p>Using posts/index.tmpl</p>
</html>
{{ end }}
```
templates/users/index.tmpl
```html
{{ define "users/index.tmpl" }}
<html><h1>
	{{ .title }}
</h1>
<p>Using users/index.tmpl</p>
</html>
{{ end }}
```

You can also use your own html template render

```go
import "html/template"

func main() {
	router := gin.Default()
	html := template.Must(template.ParseFiles("file1", "file2"))
	router.SetHTMLTemplate(html)
	router.Run(":8080")
}
```


#### Redirects

Issuing a HTTP redirect is easy:

```go
r.GET("/test", func(c *gin.Context) {
	c.Redirect(http.StatusMovedPermanently, "http://www.google.com/")
})
```
Both internal and external locations are supported.


#### Custom Middleware

```go
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()

		// Set example variable
		c.Set("example", "12345")

		// before request

		c.Next()

		// after request
		latency := time.Since(t)
		log.Print(latency)

		// access the status we are sending
		status := c.Writer.Status()
		log.Println(status)
	}
}

func main() {
	r := gin.New()
	r.Use(Logger())

	r.GET("/test", func(c *gin.Context) {
		example := c.MustGet("example").(string)

		// it would print: "12345"
		log.Println(example)
	})

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
```

#### Using BasicAuth() middleware
```go
// simulate some private data
var secrets = gin.H{
	"foo":    gin.H{"email": "foo@bar.com", "phone": "123433"},
	"austin": gin.H{"email": "austin@example.com", "phone": "666"},
	"lena":   gin.H{"email": "lena@guapa.com", "phone": "523443"},
}

func main() {
	r := gin.Default()

	// Group using gin.BasicAuth() middleware
	// gin.Accounts is a shortcut for map[string]string
	authorized := r.Group("/admin", gin.BasicAuth(gin.Accounts{
		"foo":    "bar",
		"austin": "1234",
		"lena":   "hello2",
		"manu":   "4321",
	}))

	// /admin/secrets endpoint
	// hit "localhost:8080/admin/secrets
	authorized.GET("/secrets", func(c *gin.Context) {
		// get user, it was set by the BasicAuth middleware
		user := c.MustGet(gin.AuthUserKey).(string)
		if secret, ok := secrets[user]; ok {
			c.JSON(http.StatusOK, gin.H{"user": user, "secret": secret})
		} else {
			c.JSON(http.StatusOK, gin.H{"user": user, "secret": "NO SECRET :("})
		}
	})

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
```


#### Goroutines inside a middleware
When starting inside a middleware or handler, you **SHOULD NOT** use the original context inside it, you have to use a read-only copy.

```go
func main() {
	r := gin.Default()

	r.GET("/long_async", func(c *gin.Context) {
		// create copy to be used inside the goroutine
		cCp := c.Copy()
		go func() {
			// simulate a long task with time.Sleep(). 5 seconds
			time.Sleep(5 * time.Second)

			// note that you are using the copied context "cCp", IMPORTANT
			log.Println("Done! in path " + cCp.Request.URL.Path)
		}()
	})

	r.GET("/long_sync", func(c *gin.Context) {
		// simulate a long task with time.Sleep(). 5 seconds
		time.Sleep(5 * time.Second)

		// since we are NOT using a goroutine, we do not have to copy the context
		log.Println("Done! in path " + c.Request.URL.Path)
	})

	// Listen and server on 0.0.0.0:8080
	r.Run(":8080")
}
```

#### Custom HTTP configuration

Use `http.ListenAndServe()` directly, like this:

```go
func main() {
	router := gin.Default()
	http.ListenAndServe(":8080", router)
}
```
or

```go
func main() {
	router := gin.Default()

	s := &http.Server{
		Addr:           ":8080",
		Handler:        router,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	s.ListenAndServe()
}
```

#### Graceful restart or stop

Do you want to graceful restart or stop your web server?
There are some ways this can be done.

We can use [fvbock/endless](https://github.com/fvbock/endless) to replace the default `ListenAndServe`. Refer issue [#296](https://github.com/gin-gonic/gin/issues/296) for more details.

```go
router := gin.Default()
router.GET("/", handler)
// [...]
endless.ListenAndServe(":4242", router)
```

An alternative to endless:

* [manners](https://github.com/braintree/manners): A polite Go HTTP server that shuts down gracefully.
