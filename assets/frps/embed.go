package frpc

import (
	"embed"

	"github.com/fatedier/frp/assets"
)

//go:embed *
var content embed.FS

func init() {
	assets.Register(content)
}
