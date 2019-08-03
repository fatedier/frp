package frp

import (
	"github.com/fatedier/frp/client"
	"github.com/fatedier/frp/cmd/frpc/sub"
	"log"
)

type FRPCService struct {
	*client.Service
	l                   FRPCClosedListener
	proxyFailedListener ProxyFailedListener
}

type FRPCClosedListener interface {
	OnClosed(msg string)
}

type ProxyFailedListener interface {
	OnProxyFailed()
}

func (frp *FRPCService) OnClosed(msg string) {
	log.Printf("OnClosed() l = %v", frp.l)
	if frp.l != nil {
		frp.l.OnClosed(msg)
	}
}

func (frp *FRPCService) SetFRPCCloseListener(listener FRPCClosedListener) {
	frp.l = listener
	frp.SetOnCloseListener(frp)
}

func (frp *FRPCService) onProxyFailed(err error) {
	if frp.proxyFailedListener != nil {
		frp.proxyFailedListener.OnProxyFailed()
	}
}

func (frp *FRPCService) SetProxyFailedListener(listener ProxyFailedListener) {
	frp.proxyFailedListener = listener
	frp.SetProxyFailedFunc(frp.onProxyFailed)
}

func (frp *FRPCService) SetReConnectByCount(reConnectByCount bool) {
	frp.ReConnectByCount = reConnectByCount
}

func NewFrpcServiceWithPath(path string) (*FRPCService, error) {
	svr, err := sub.NewService(path)
	if err != nil {
		return nil, err
	}
	return &FRPCService{Service: svr}, nil
}
