package frps

import (
	"embed"

	"github.com/fatedier/frp/assets"
)

//go:embed dist
var EmbedFS embed.FS

func init() {
	assets.Register(EmbedFS)
}
