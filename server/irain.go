package server

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type IRainApiRestult struct {
	Code    int
	Message string
}

func IRainRespone(w http.ResponseWriter, code int, msg string) {
	ret := IRainApiRestult{
		Code:    code,
		Message: msg,
	}
	b, _ := json.Marshal(ret)
	w.WriteHeader(code)
	w.Write(b)
}

func irainSign(ip string, timestamp int64) string {
	// 得到动态密钥
	key := time.Now().Format("20060102") + "irainkey"
	src := fmt.Sprintf("%s%d%s", key, timestamp, ip)
	return fmt.Sprintf("%x", md5.Sum([]byte(src)))
}

type IRainIPPool struct {
	mux  sync.RWMutex
	list map[string]time.Time
}

var globalIRainIPPool = &IRainIPPool{list: make(map[string]time.Time)}

func (p *IRainIPPool) Put(ip string) {
	// 过期时间为当前时间后的半小时
	p.mux.Lock()
	defer p.mux.Unlock()
	p.list[ip] = time.Now().Add(time.Minute * 30)
}

func (p *IRainIPPool) Check(ip string) bool {
	p.mux.RLock()
	defer p.mux.RUnlock()
	if v, ok := p.list[ip]; ok {
		if time.Now().Before(v) {
			return true
		}
	}
	return false
}

// IrainToken 获取可以访问的客户端地址
func IrainToken(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	var (
		clientIP     = r.PostFormValue("ip")
		timestamp, _ = strconv.ParseInt(r.PostFormValue("timestamp"), 10, 64)
		sign         = r.PostFormValue("sign")
	)
	if sign == "" || clientIP == "" {
		IRainRespone(w, 400, "参数错误")
		return
	}
	if (time.Now().Unix() - timestamp) > 60*30 {
		IRainRespone(w, 403, "请求已过期")
		return
	}
	if irainSign(clientIP, timestamp) != sign {
		IRainRespone(w, 403, "签名错误")
		return
	}
	globalIRainIPPool.Put(clientIP)
	IRainRespone(w, 0, "ok")
}
