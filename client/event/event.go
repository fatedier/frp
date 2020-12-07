package event

import (
	"errors"

	"github.com/fatedier/frp/pkg/msg"
)

type Type int

const (
	EvStartProxy Type = iota
	EvCloseProxy
)

var (
	ErrPayloadType = errors.New("error payload type")
)

type Handler func(evType Type, payload interface{}) error

type StartProxyPayload struct {
	NewProxyMsg *msg.NewProxy
}

type CloseProxyPayload struct {
	CloseProxyMsg *msg.CloseProxy
}
