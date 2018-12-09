package event

import (
	"errors"

	"github.com/fatedier/frp/models/msg"
)

type EventType int

const (
	EvStartProxy EventType = iota
	EvCloseProxy
)

var (
	ErrPayloadType = errors.New("error payload type")
)

type EventHandler func(evType EventType, payload interface{}) error

type StartProxyPayload struct {
	NewProxyMsg *msg.NewProxy
}

type CloseProxyPayload struct {
	CloseProxyMsg *msg.CloseProxy
}
