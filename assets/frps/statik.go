package frps

import (
	"embed"
	"github.com/fatedier/frp/assets"
	"io/fs"
)

//go:embed static
var staticFiles embed.FS

func init() {
	assets.StaticFiles, _ = fs.Sub(staticFiles, "static")
}
