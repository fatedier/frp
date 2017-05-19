package proxy

import (
	"bufio"
	"encoding/json"
	"os"
)

// Config 保存代理服务器的配置
type Config struct {
	Port string `json:"port"`
	Auth bool   `json:"auth"`

	User map[string]string `json:"user"`
}

// 从指定json文件读取config配置
func (c *Config) GetConfig(filename string) error {

	configFile, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer configFile.Close()

	br := bufio.NewReader(configFile)
	err = json.NewDecoder(br).Decode(c)
	if err != nil {
		return err
	}
	return nil
}
