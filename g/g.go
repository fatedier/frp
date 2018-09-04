package g

import (
	"github.com/fatedier/frp/models/config"
)

var (
	GlbClientCfg *ClientCfg
	GlbServerCfg *ServerCfg
)

func init() {
	GlbClientCfg = &ClientCfg{
		ClientCommonConf: *config.GetDefaultClientConf(),
	}
	GlbServerCfg = &ServerCfg{
		ServerCommonConf: *config.GetDefaultServerConf(),
	}
}

type ClientCfg struct {
	config.ClientCommonConf

	CfgFile       string
	ServerUdpPort int // this is configured by login response from frps
}

type ServerCfg struct {
	config.ServerCommonConf

	CfgFile string
}
