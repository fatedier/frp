package ssdb

import (
	"strconv"
	"testing"
	"time"

	"github.com/astaxie/beego/cache"
)

func TestSsdbcacheCache(t *testing.T) {
	ssdb, err := cache.NewCache("ssdb", `{"conn": "127.0.0.1:8888"}`)
	if err != nil {
		t.Error("init err")
	}

	// test put and exist
	if ssdb.IsExist("ssdb") {
		t.Error("check err")
	}
	timeoutDuration := 10 * time.Second
	//timeoutDuration := -10*time.Second   if timeoutDuration is negtive,it means permanent
	if err = ssdb.Put("ssdb", "ssdb", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if !ssdb.IsExist("ssdb") {
		t.Error("check err")
	}

	// Get test done
	if err = ssdb.Put("ssdb", "ssdb", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}

	if v := ssdb.Get("ssdb"); v != "ssdb" {
		t.Error("get Error")
	}

	//inc/dec test done
	if err = ssdb.Put("ssdb", "2", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if err = ssdb.Incr("ssdb"); err != nil {
		t.Error("incr Error", err)
	}

	if v, err := strconv.Atoi(ssdb.Get("ssdb").(string)); err != nil || v != 3 {
		t.Error("get err")
	}

	if err = ssdb.Decr("ssdb"); err != nil {
		t.Error("decr error")
	}

	// test del
	if err = ssdb.Put("ssdb", "3", timeoutDuration); err != nil {
		t.Error("set Error", err)
	}
	if v, err := strconv.Atoi(ssdb.Get("ssdb").(string)); err != nil || v != 3 {
		t.Error("get err")
	}
	if err := ssdb.Delete("ssdb"); err == nil {
		if ssdb.IsExist("ssdb") {
			t.Error("delete err")
		}
	}

	//test string
	if err = ssdb.Put("ssdb", "ssdb", -10*time.Second); err != nil {
		t.Error("set Error", err)
	}
	if !ssdb.IsExist("ssdb") {
		t.Error("check err")
	}
	if v := ssdb.Get("ssdb").(string); v != "ssdb" {
		t.Error("get err")
	}

	//test GetMulti done
	if err = ssdb.Put("ssdb1", "ssdb1", -10*time.Second); err != nil {
		t.Error("set Error", err)
	}
	if !ssdb.IsExist("ssdb1") {
		t.Error("check err")
	}
	vv := ssdb.GetMulti([]string{"ssdb", "ssdb1"})
	if len(vv) != 2 {
		t.Error("getmulti error")
	}
	if vv[0].(string) != "ssdb" {
		t.Error("getmulti error")
	}
	if vv[1].(string) != "ssdb1" {
		t.Error("getmulti error")
	}

	// test clear all done
	if err = ssdb.ClearAll(); err != nil {
		t.Error("clear all err")
	}
	if ssdb.IsExist("ssdb") || ssdb.IsExist("ssdb1") {
		t.Error("check err")
	}
}
