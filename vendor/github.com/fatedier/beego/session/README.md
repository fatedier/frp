session
==============

session is a Go session manager. It can use many session providers. Just like the `database/sql` and `database/sql/driver`.

## How to install?

	go get github.com/astaxie/beego/session


## What providers are supported?

As of now this session manager support memory, file, Redis and MySQL.


## How to use it?

First you must import it

	import (
		"github.com/astaxie/beego/session"
	)

Then in you web app init the global session manager
	
	var globalSessions *session.Manager

* Use **memory** as provider:

		func init() {
			globalSessions, _ = session.NewManager("memory", `{"cookieName":"gosessionid","gclifetime":3600}`)
			go globalSessions.GC()
		}

* Use **file** as provider, the last param is the path where you want file to be stored:

		func init() {
			globalSessions, _ = session.NewManager("file",`{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"./tmp"}`)
			go globalSessions.GC()
		}

* Use **Redis** as provider, the last param is the Redis conn address,poolsize,password:

		func init() {
			globalSessions, _ = session.NewManager("redis", `{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"127.0.0.1:6379,100,astaxie"}`)
			go globalSessions.GC()
		}
		
* Use **MySQL** as provider, the last param is the DSN, learn more from [mysql](https://github.com/go-sql-driver/mysql#dsn-data-source-name):

		func init() {
			globalSessions, _ = session.NewManager(
				"mysql", `{"cookieName":"gosessionid","gclifetime":3600,"ProviderConfig":"username:password@protocol(address)/dbname?param=value"}`)
			go globalSessions.GC()
		}

* Use **Cookie** as provider:

		func init() {
			globalSessions, _ = session.NewManager(
				"cookie", `{"cookieName":"gosessionid","enableSetCookie":false,"gclifetime":3600,"ProviderConfig":"{\"cookieName\":\"gosessionid\",\"securityKey\":\"beegocookiehashkey\"}"}`)
			go globalSessions.GC()
		}


Finally in the handlerfunc you can use it like this

	func login(w http.ResponseWriter, r *http.Request) {
		sess := globalSessions.SessionStart(w, r)
		defer sess.SessionRelease(w)
		username := sess.Get("username")
		fmt.Println(username)
		if r.Method == "GET" {
			t, _ := template.ParseFiles("login.gtpl")
			t.Execute(w, nil)
		} else {
			fmt.Println("username:", r.Form["username"])
			sess.Set("username", r.Form["username"])
			fmt.Println("password:", r.Form["password"])
		}
	}


## How to write own provider?

When you develop a web app, maybe you want to write own provider because you must meet the requirements.

Writing a provider is easy. You only need to define two struct types 
(Session and Provider), which satisfy the interface definition. 
Maybe you will find the **memory** provider is a good example.

	type SessionStore interface {
		Set(key, value interface{}) error     //set session value
		Get(key interface{}) interface{}      //get session value
		Delete(key interface{}) error         //delete session value
		SessionID() string                    //back current sessionID
		SessionRelease(w http.ResponseWriter) // release the resource & save data to provider & return the data
		Flush() error                         //delete all data
	}
	
	type Provider interface {
		SessionInit(gclifetime int64, config string) error
		SessionRead(sid string) (SessionStore, error)
		SessionExist(sid string) bool
		SessionRegenerate(oldsid, sid string) (SessionStore, error)
		SessionDestroy(sid string) error
		SessionAll() int //get all active session
		SessionGC()
	}


## LICENSE

BSD License http://creativecommons.org/licenses/BSD/
