package logs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// JLWriter implements beego LoggerInterface and is used to send jiaoliao webhook
type JLWriter struct {
	AuthorName  string `json:"authorname"`
	Title       string `json:"title"`
	WebhookURL  string `json:"webhookurl"`
	RedirectURL string `json:"redirecturl,omitempty"`
	ImageURL    string `json:"imageurl,omitempty"`
	Level       int    `json:"level"`
}

// newJLWriter create jiaoliao writer.
func newJLWriter() Logger {
	return &JLWriter{Level: LevelTrace}
}

// Init JLWriter with json config string
func (s *JLWriter) Init(jsonconfig string) error {
	err := json.Unmarshal([]byte(jsonconfig), s)
	if err != nil {
		return err
	}
	return nil
}

// WriteMsg write message in smtp writer.
// it will send an email with subject and only this message.
func (s *JLWriter) WriteMsg(when time.Time, msg string, level int) error {
	if level > s.Level {
		return nil
	}

	text := fmt.Sprintf("%s %s", when.Format("2006-01-02 15:04:05"), msg)

	form := url.Values{}
	form.Add("authorName", s.AuthorName)
	form.Add("title", s.Title)
	form.Add("text", text)
	if s.RedirectURL != "" {
		form.Add("redirectUrl", s.RedirectURL)
	}
	if s.ImageURL != "" {
		form.Add("imageUrl", s.ImageURL)
	}

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
func (s *JLWriter) Flush() {
	return
}

// Destroy implementing method. empty.
func (s *JLWriter) Destroy() {
	return
}

func init() {
	Register(AdapterJianLiao, newJLWriter)
}
