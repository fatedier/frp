package frp

import "github.com/fatedier/frp/pkg/util/log"

type FrpLogListener interface {
	Log(log string)
	Location() string
}

// type FrpLogListener struct {
// 	name string
// }

// func (l *FrpLogListener) Log(log string) {
// }

func SetFrpLogListener(l FrpLogListener) {
	log.AppendListener(l)
}
