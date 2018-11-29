package g

import (
	"github.com/fatedier/frp/models/config"
)

var (
	GlbClientCfg *ClientCfg
	GlbServerCfg *ServerCfg
	GlbServerSubSectionMap map[string]*config.SubServerSectionConf
)

func init() {
	GlbClientCfg = &ClientCfg{
		ClientCommonConf: *config.GetDefaultClientConf(),
	}
	GlbServerCfg = &ServerCfg{
		ServerSectionConf: *config.GetDefaultServerConf(),
	}
	GlbServerSubSectionMap = make(map[string]*config.SubServerSectionConf)
}

type ClientCfg struct {
	config.ClientCommonConf

	CfgFile       string
	ServerUdpPort int // this is configured by login response from frps
}

type ServerCfg struct {
	config.ServerSectionConf

	CfgFile string
}
