package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/test/e2e/pkg/utils"
)

type Client struct {
	address  string
	authUser string
	authPwd  string
}

func New(host string, port int) *Client {
	return &Client{
		address: net.JoinHostPort(host, strconv.Itoa(port)),
	}
}

func (c *Client) SetAuth(user, pwd string) {
	c.authUser = user
	c.authPwd = pwd
}

func (c *Client) GetProxyStatus(name string) (*client.ProxyStatusResp, error) {
	req, err := http.NewRequest("GET", "http://"+c.address+"/api/status", nil)
	if err != nil {
		return nil, err
	}
	content, err := c.do(req)
	if err != nil {
		return nil, err
	}
	allStatus := &client.StatusResp{}
	if err = json.Unmarshal([]byte(content), &allStatus); err != nil {
		return nil, fmt.Errorf("unmarshal http response error: %s", strings.TrimSpace(content))
	}
	for _, s := range allStatus.TCP {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.UDP {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.HTTP {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.HTTPS {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.STCP {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.XTCP {
		if s.Name == name {
			return &s, nil
		}
	}
	for _, s := range allStatus.SUDP {
		if s.Name == name {
			return &s, nil
		}
	}
	return nil, fmt.Errorf("no proxy status found")
}

func (c *Client) Reload() error {
	req, err := http.NewRequest("GET", "http://"+c.address+"/api/reload", nil)
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

func (c *Client) GetConfig() (string, error) {
	req, err := http.NewRequest("GET", "http://"+c.address+"/api/config", nil)
	if err != nil {
		return "", err
	}
	return c.do(req)
}

func (c *Client) UpdateConfig(content string) error {
	req, err := http.NewRequest("PUT", "http://"+c.address+"/api/config", strings.NewReader(content))
	if err != nil {
		return err
	}
	_, err = c.do(req)
	return err
}

func (c *Client) setAuthHeader(req *http.Request) {
	if c.authUser != "" || c.authPwd != "" {
		req.Header.Set("Authorization", utils.BasicAuth(c.authUser, c.authPwd))
	}
}

func (c *Client) do(req *http.Request) (string, error) {
	c.setAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("api status code [%d]", resp.StatusCode)
	}
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(buf), nil
}
