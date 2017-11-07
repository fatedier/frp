package alils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"strconv"

	lz4 "github.com/cloudflare/golz4"
	"github.com/gogo/protobuf/proto"
)

type LogStore struct {
	Name       string `json:"logstoreName"`
	TTL        int
	ShardCount int

	CreateTime     uint32
	LastModifyTime uint32

	project *LogProject
}

type Shard struct {
	ShardID int `json:"shardID"`
}

// ListShards returns shard id list of this logstore.
func (s *LogStore) ListShards() (shardIDs []int, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/logstores/%v/shards", s.Name)
	r, err := request(s.project, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to list logstore")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	var shards []*Shard
	err = json.Unmarshal(buf, &shards)
	if err != nil {
		return
	}

	for _, v := range shards {
		shardIDs = append(shardIDs, v.ShardID)
	}
	return
}

// PutLogs put logs into logstore.
// The callers should transform user logs into LogGroup.
func (s *LogStore) PutLogs(lg *LogGroup) (err error) {
	body, err := proto.Marshal(lg)
	if err != nil {
		return
	}

	// Compresse body with lz4
	out := make([]byte, lz4.CompressBound(body))
	n, err := lz4.Compress(body, out)
	if err != nil {
		return
	}

	h := map[string]string{
		"x-sls-compresstype": "lz4",
		"x-sls-bodyrawsize":  fmt.Sprintf("%v", len(body)),
		"Content-Type":       "application/x-protobuf",
	}

	uri := fmt.Sprintf("/logstores/%v", s.Name)
	r, err := request(s.project, "POST", uri, h, out[:n])
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to put logs")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}
	return
}

// GetCursor gets log cursor of one shard specified by shardId.
// The from can be in three form: a) unix timestamp in seccond, b) "begin", c) "end".
// For more detail please read: http://gitlab.alibaba-inc.com/sls/doc/blob/master/api/shard.md#logstore
func (s *LogStore) GetCursor(shardId int, from string) (cursor string, err error) {
	h := map[string]string{
		"x-sls-bodyrawsize": "0",
	}

	uri := fmt.Sprintf("/logstores/%v/shards/%v?type=cursor&from=%v",
		s.Name, shardId, from)

	r, err := request(s.project, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to get cursor")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	type Body struct {
		Cursor string
	}
	body := &Body{}

	err = json.Unmarshal(buf, body)
	if err != nil {
		return
	}
	cursor = body.Cursor
	return
}

// GetLogsBytes gets logs binary data from shard specified by shardId according cursor.
// The logGroupMaxCount is the max number of logGroup could be returned.
// The nextCursor is the next curosr can be used to read logs at next time.
func (s *LogStore) GetLogsBytes(shardId int, cursor string,
	logGroupMaxCount int) (out []byte, nextCursor string, err error) {

	h := map[string]string{
		"x-sls-bodyrawsize": "0",
		"Accept":            "application/x-protobuf",
		"Accept-Encoding":   "lz4",
	}

	uri := fmt.Sprintf("/logstores/%v/shards/%v?type=logs&cursor=%v&count=%v",
		s.Name, shardId, cursor, logGroupMaxCount)

	r, err := request(s.project, "GET", uri, h, nil)
	if err != nil {
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}

	if r.StatusCode != http.StatusOK {
		errMsg := &errorMessage{}
		err = json.Unmarshal(buf, errMsg)
		if err != nil {
			err = fmt.Errorf("failed to get cursor")
			dump, _ := httputil.DumpResponse(r, true)
			fmt.Println(dump)
			return
		}
		err = fmt.Errorf("%v:%v", errMsg.Code, errMsg.Message)
		return
	}

	v, ok := r.Header["X-Sls-Compresstype"]
	if !ok || len(v) == 0 {
		err = fmt.Errorf("can't find 'x-sls-compresstype' header")
		return
	}
	if v[0] != "lz4" {
		err = fmt.Errorf("unexpected compress type:%v", v[0])
		return
	}

	v, ok = r.Header["X-Sls-Cursor"]
	if !ok || len(v) == 0 {
		err = fmt.Errorf("can't find 'x-sls-cursor' header")
		return
	}
	nextCursor = v[0]

	v, ok = r.Header["X-Sls-Bodyrawsize"]
	if !ok || len(v) == 0 {
		err = fmt.Errorf("can't find 'x-sls-bodyrawsize' header")
		return
	}
	bodyRawSize, err := strconv.Atoi(v[0])
	if err != nil {
		return
	}

	out = make([]byte, bodyRawSize)
	err = lz4.Uncompress(buf, out)
	if err != nil {
		return
	}

	return
}

// LogsBytesDecode decodes logs binary data retruned by GetLogsBytes API
func LogsBytesDecode(data []byte) (gl *LogGroupList, err error) {

	gl = &LogGroupList{}
	err = proto.Unmarshal(data, gl)
	if err != nil {
		return
	}

	return
}

// GetLogs gets logs from shard specified by shardId according cursor.
// The logGroupMaxCount is the max number of logGroup could be returned.
// The nextCursor is the next curosr can be used to read logs at next time.
func (s *LogStore) GetLogs(shardId int, cursor string,
	logGroupMaxCount int) (gl *LogGroupList, nextCursor string, err error) {

	out, nextCursor, err := s.GetLogsBytes(shardId, cursor, logGroupMaxCount)
	if err != nil {
		return
	}

	gl, err = LogsBytesDecode(out)
	if err != nil {
		return
	}

	return
}
