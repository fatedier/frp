package beego

import (
	"fmt"
	"testing"
)

func TestList_01(t *testing.T) {
	m := make(map[string]interface{})
	list("BConfig", BConfig, m)
	t.Log(m)
	om := oldMap()
	for k, v := range om {
		if fmt.Sprint(m[k]) != fmt.Sprint(v) {
			t.Log(k, "old-key", v, "new-key", m[k])
			t.FailNow()
		}
	}
}

func oldMap() map[string]interface{} {
	m := make(map[string]interface{})
	m["BConfig.AppName"] = BConfig.AppName
	m["BConfig.RunMode"] = BConfig.RunMode
	m["BConfig.RouterCaseSensitive"] = BConfig.RouterCaseSensitive
	m["BConfig.ServerName"] = BConfig.ServerName
	m["BConfig.RecoverPanic"] = BConfig.RecoverPanic
	m["BConfig.CopyRequestBody"] = BConfig.CopyRequestBody
	m["BConfig.EnableGzip"] = BConfig.EnableGzip
	m["BConfig.MaxMemory"] = BConfig.MaxMemory
	m["BConfig.EnableErrorsShow"] = BConfig.EnableErrorsShow
	m["BConfig.Listen.Graceful"] = BConfig.Listen.Graceful
	m["BConfig.Listen.ServerTimeOut"] = BConfig.Listen.ServerTimeOut
	m["BConfig.Listen.ListenTCP4"] = BConfig.Listen.ListenTCP4
	m["BConfig.Listen.EnableHTTP"] = BConfig.Listen.EnableHTTP
	m["BConfig.Listen.HTTPAddr"] = BConfig.Listen.HTTPAddr
	m["BConfig.Listen.HTTPPort"] = BConfig.Listen.HTTPPort
	m["BConfig.Listen.EnableHTTPS"] = BConfig.Listen.EnableHTTPS
	m["BConfig.Listen.HTTPSAddr"] = BConfig.Listen.HTTPSAddr
	m["BConfig.Listen.HTTPSPort"] = BConfig.Listen.HTTPSPort
	m["BConfig.Listen.HTTPSCertFile"] = BConfig.Listen.HTTPSCertFile
	m["BConfig.Listen.HTTPSKeyFile"] = BConfig.Listen.HTTPSKeyFile
	m["BConfig.Listen.EnableAdmin"] = BConfig.Listen.EnableAdmin
	m["BConfig.Listen.AdminAddr"] = BConfig.Listen.AdminAddr
	m["BConfig.Listen.AdminPort"] = BConfig.Listen.AdminPort
	m["BConfig.Listen.EnableFcgi"] = BConfig.Listen.EnableFcgi
	m["BConfig.Listen.EnableStdIo"] = BConfig.Listen.EnableStdIo
	m["BConfig.WebConfig.AutoRender"] = BConfig.WebConfig.AutoRender
	m["BConfig.WebConfig.EnableDocs"] = BConfig.WebConfig.EnableDocs
	m["BConfig.WebConfig.FlashName"] = BConfig.WebConfig.FlashName
	m["BConfig.WebConfig.FlashSeparator"] = BConfig.WebConfig.FlashSeparator
	m["BConfig.WebConfig.DirectoryIndex"] = BConfig.WebConfig.DirectoryIndex
	m["BConfig.WebConfig.StaticDir"] = BConfig.WebConfig.StaticDir
	m["BConfig.WebConfig.StaticExtensionsToGzip"] = BConfig.WebConfig.StaticExtensionsToGzip
	m["BConfig.WebConfig.TemplateLeft"] = BConfig.WebConfig.TemplateLeft
	m["BConfig.WebConfig.TemplateRight"] = BConfig.WebConfig.TemplateRight
	m["BConfig.WebConfig.ViewsPath"] = BConfig.WebConfig.ViewsPath
	m["BConfig.WebConfig.EnableXSRF"] = BConfig.WebConfig.EnableXSRF
	m["BConfig.WebConfig.XSRFExpire"] = BConfig.WebConfig.XSRFExpire
	m["BConfig.WebConfig.Session.SessionOn"] = BConfig.WebConfig.Session.SessionOn
	m["BConfig.WebConfig.Session.SessionProvider"] = BConfig.WebConfig.Session.SessionProvider
	m["BConfig.WebConfig.Session.SessionName"] = BConfig.WebConfig.Session.SessionName
	m["BConfig.WebConfig.Session.SessionGCMaxLifetime"] = BConfig.WebConfig.Session.SessionGCMaxLifetime
	m["BConfig.WebConfig.Session.SessionProviderConfig"] = BConfig.WebConfig.Session.SessionProviderConfig
	m["BConfig.WebConfig.Session.SessionCookieLifeTime"] = BConfig.WebConfig.Session.SessionCookieLifeTime
	m["BConfig.WebConfig.Session.SessionAutoSetCookie"] = BConfig.WebConfig.Session.SessionAutoSetCookie
	m["BConfig.WebConfig.Session.SessionDomain"] = BConfig.WebConfig.Session.SessionDomain
	m["BConfig.WebConfig.Session.SessionDisableHTTPOnly"] = BConfig.WebConfig.Session.SessionDisableHTTPOnly
	m["BConfig.Log.AccessLogs"] = BConfig.Log.AccessLogs
	m["BConfig.Log.FileLineNum"] = BConfig.Log.FileLineNum
	m["BConfig.Log.Outputs"] = BConfig.Log.Outputs
	return m
}
