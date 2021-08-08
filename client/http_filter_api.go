package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/fatedier/frp/pkg/plugin/interceptor"
	"github.com/fatedier/frp/pkg/util/cache"
	"github.com/fatedier/frp/pkg/util/log"
	"github.com/gorilla/mux"
)

func (svr *Service) httpFilterList(w http.ResponseWriter, r *http.Request) {
	keys := cache.DefaultCache.Keys()
	if len(keys) <= 0 {
		log.Debug("there is no http stream be cached")
		w.WriteHeader(200)
		return
	}

	data, _ := json.Marshal(keys)

	w.WriteHeader(200)
	w.Write(data)
}

func (svr *Service) httpFilterGetRaw(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dataRaw, ok := cache.DefaultCache.Get(id)
	if !ok {
		log.Warn("key not found: %v", id)
		w.WriteHeader(404)
		return
	}
	data, _ := json.Marshal(dataRaw)

	w.WriteHeader(200)
	w.Write(data)
}

func (svr *Service) httpFilterReplay(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	dataRaw, ok := cache.DefaultCache.Get(id)
	if !ok {
		log.Warn("key not found: %v", id)
		w.WriteHeader(404)
		return
	}

	pair, ok := dataRaw.(interceptor.Pair)
	if !ok {
		log.Warn("data type not match")
		w.WriteHeader(500)
		return
	}

	req, _ := http.NewRequest(pair.Req.Method, pair.Req.URL, bytes.NewBuffer(pair.Req.Body))
	req.Header = pair.Req.Header
	req.Host = pair.Req.Host

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warn("replay request: %v got error: %v", pair.Req, err)
		w.WriteHeader(500)
		return
	}
	defer resp.Body.Close()

	data, _ := ioutil.ReadAll(resp.Body)
	log.Debug("replay get response body: %v", string(data))

	w.WriteHeader(200)
}

func (svr *Service) httpFilterRemove(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	cache.DefaultCache.Remove(id)

	w.WriteHeader(200)
}

func (svr *Service) httpFilterClear(w http.ResponseWriter, r *http.Request) {
	cache.DefaultCache.Purge()
	w.WriteHeader(200)
}
