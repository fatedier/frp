package g

import (
	"github.com/fatedier/frp/models/config"
)

var (
	GlbClientCfg *ClientCfg
)

func init() {
	GlbClientCfg = &ClientCfg{
		ClientCommonConf: *config.GetDefaultClientConf(),
	}
}

type ClientCfg struct {
	config.ClientCommonConf

	CfgFile       string
	ServerUdpPort int // this is configured by login response from frps
}
