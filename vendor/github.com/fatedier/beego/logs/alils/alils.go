package alils

import (
	"encoding/json"
	"github.com/astaxie/beego/logs"
	"github.com/gogo/protobuf/proto"
	"strings"
	"sync"
	"time"
)

const (
	CacheSize int    = 64
	Delimiter string = "##"
)

type AliLSConfig struct {
	Project   string   `json:"project"`
	Endpoint  string   `json:"endpoint"`
	KeyID     string   `json:"key_id"`
	KeySecret string   `json:"key_secret"`
	LogStore  string   `json:"log_store"`
	Topics    []string `json:"topics"`
	Source    string   `json:"source"`
	Level     int      `json:"level"`
	FlushWhen int      `json:"flush_when"`
}

// aliLSWriter implements LoggerInterface.
// it writes messages in keep-live tcp connection.
type aliLSWriter struct {
	store    *LogStore
	group    []*LogGroup
	withMap  bool
	groupMap map[string]*LogGroup
	lock     *sync.Mutex
	AliLSConfig
}

// 创建提供Logger接口的日志服务
func NewAliLS() logs.Logger {
	alils := new(aliLSWriter)
	alils.Level = logs.LevelTrace
	return alils
}

// 读取配置
// 初始化必要的数据结构
func (c *aliLSWriter) Init(jsonConfig string) (err error) {

	json.Unmarshal([]byte(jsonConfig), c)

	if c.FlushWhen > CacheSize {
		c.FlushWhen = CacheSize
	}

	// 初始化Project
	prj := &LogProject{
		Name:            c.Project,
		Endpoint:        c.Endpoint,
		AccessKeyId:     c.KeyID,
		AccessKeySecret: c.KeySecret,
	}

	// 获取logstore
	c.store, err = prj.GetLogStore(c.LogStore)
	if err != nil {
		return err
	}

	// 创建默认Log Group
	c.group = append(c.group, &LogGroup{
		Topic:  proto.String(""),
		Source: proto.String(c.Source),
		Logs:   make([]*Log, 0, c.FlushWhen),
	})

	// 创建其它Log Group
	c.groupMap = make(map[string]*LogGroup)
	for _, topic := range c.Topics {

		lg := &LogGroup{
			Topic:  proto.String(topic),
			Source: proto.String(c.Source),
			Logs:   make([]*Log, 0, c.FlushWhen),
		}

		c.group = append(c.group, lg)
		c.groupMap[topic] = lg
	}

	if len(c.group) == 1 {
		c.withMap = false
	} else {
		c.withMap = true
	}

	c.lock = &sync.Mutex{}

	return nil
}

// WriteMsg write message in connection.
// if connection is down, try to re-connect.
func (c *aliLSWriter) WriteMsg(when time.Time, msg string, level int) (err error) {

	if level > c.Level {
		return nil
	}

	var topic string
	var content string
	var lg *LogGroup
	if c.withMap {

		// 解析出Topic，并匹配LogGroup
		strs := strings.SplitN(msg, Delimiter, 2)
		if len(strs) == 2 {
			pos := strings.LastIndex(strs[0], " ")
			topic = strs[0][pos+1 : len(strs[0])]
			content = strs[0][0:pos] + strs[1]
			lg = c.groupMap[topic]
		}

		// 默认发到空Topic
		if lg == nil {
			topic = ""
			content = msg
			lg = c.group[0]
		}
	} else {
		topic = ""
		content = msg
		lg = c.group[0]
	}

	// 生成日志
	c1 := &Log_Content{
		Key:   proto.String("msg"),
		Value: proto.String(content),
	}

	l := &Log{
		Time: proto.Uint32(uint32(when.Unix())), // 填写日志时间
		Contents: []*Log_Content{
			c1,
		},
	}

	c.lock.Lock()
	lg.Logs = append(lg.Logs, l)
	c.lock.Unlock()

	// 满足条件则Flush
	if len(lg.Logs) >= c.FlushWhen {
		c.flush(lg)
	}

	return nil
}

// Flush implementing method. empty.
func (c *aliLSWriter) Flush() {

	// flush所有group
	for _, lg := range c.group {
		c.flush(lg)
	}
}

// Destroy destroy connection writer and close tcp listener.
func (c *aliLSWriter) Destroy() {
}

func (c *aliLSWriter) flush(lg *LogGroup) {

	c.lock.Lock()
	defer c.lock.Unlock()

	// 把以上的LogGroup推送到SLS服务器，
	// SLS服务器会根据该logstore的shard个数自动进行负载均衡。
	err := c.store.PutLogs(lg)
	if err != nil {
		return
	}

	lg.Logs = make([]*Log, 0, c.FlushWhen)
}

func init() {
	logs.Register(logs.AdapterAliLS, NewAliLS)
}
