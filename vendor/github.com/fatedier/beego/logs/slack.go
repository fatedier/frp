package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// SLACKWriter implements beego LoggerInterface and is used to send jiaoliao webhook
type SLACKWriter struct {
	WebhookURL string `json:"webhookurl"`
	Level      int    `json:"level"`
}

// newSLACKWriter create jiaoliao writer.
func newSLACKWriter() Logger {
	return &SLACKWriter{Level: LevelTrace}
}

// Init SLACKWriter with json config string
func (s *SLACKWriter) Init(jsonconfig string) error {
	err := json.Unmarshal([]byte(jsonconfig), s)
	if err != nil {
		return err
	}
	return nil
}

// WriteMsg write message in smtp writer.
// it will send an email with subject and only this message.
func (s *SLACKWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > s.Level {
		return nil
	}

	text := fmt.Sprintf("{\"text\": \"%s %s\"}", when.Format("2006-01-02 15:04:05"), msg)

	form := url.Values{}
	form.Add("payload", text)

	resp, err := http.PostForm(s.WebhookURL, form)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Post webhook failed %s %d", resp.Status, resp.StatusCode)
	}
	return nil
}

// Flush implementing method. empty.
func (s *SLACKWriter) Flush() {
	return
}

// Destroy implementing method. empty.
func (s *SLACKWriter) Destroy() {
	return
}

func init() {
	Register(AdapterSlack, newSLACKWriter)
}
