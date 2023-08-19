package server

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/fatedier/frp/pkg/msg"
	"github.com/parnurzeal/gorequest"
)

type Data struct {
	// ClientIP is the IP address of the connected proxy
	ClientIP string `json:"client_ip"`
	// RemotePort is the Port number of the proxy
	RemotePort int `json:"remote_port"`
	// LocalAddr is the IP address of this frps instance
	LocalAddr string `json:"local_addr"`
	// ProxyName is the name of the connected proxy
	ProxyName string `json:"proxy_name"`
	// ProxyType is the type of the connected proxy (tcp or ssh)
	ProxyType string `json:"proxy_type"`
	// Headers is a map of the connected proxy headers
	Headers map[string]string `json:"headers"`
	// Os is the operating system the frpc is running on
	Os string `json:"os"`
	// Arch is the system architecture of the frpc
	Arch string `json:"arch"`
	// Version is the frpc proxy version
	Version string `json:"version"`
	// Hostname is the name of the frpc machine
	Hostname string `json:"hostname"`
}

type Payload struct {
	// Event is the name of the webhook event (offline or online)
	Event string `json:"event"`
	// UniqueID is the ID of the proxy
	UniqueID string `json:"unique_id"`
	// Data is a struct containing the information of the connected proxy
	Data *Data `json:"data"`
}

type Webhook struct {
	endpoint    string
	client      *gorequest.SuperAgent
	connections map[string]Payload
}

// NewWebhook returns an instance of the webhook object for each connected proxy
func NewWebhook(endpoint string) *Webhook {
	return &Webhook{
		endpoint:    endpoint,
		client:      gorequest.New(),
		connections: map[string]Payload{},
	}
}

// OnConnect is called when a proxy gets connected
func (w *Webhook) OnConnect(conn net.Conn, m *msg.NewProxy, l *msg.Login) string {
	uniqueID := l.RunID
	d := &Data{
		ClientIP:   conn.RemoteAddr().String(),
		LocalAddr:  conn.LocalAddr().String(),
		ProxyName:  m.ProxyName,
		ProxyType:  m.ProxyType,
		RemotePort: m.RemotePort,
		Headers:    m.Headers,
		Os:         l.Os,
		Arch:       l.Arch,
		Version:    l.Version,
		Hostname:   l.Hostname,
	}

	p := Payload{
		Event:    "online",
		UniqueID: uniqueID,
		Data:     d,
	}

	w.connections[uniqueID] = p
	w.send(p)

	return uniqueID
}

// OnDisconnect is called when a proxy disconnects
func (w *Webhook) OnDisconnect(id string) {
	if p, ok := w.connections[id]; ok {
		p.Event = "offline"
		w.send(p)
	}
}

func (w *Webhook) send(p Payload) {
	b, err := json.Marshal(&p)
	if err != nil {
		fmt.Printf("error marshalling payload: %s", err.Error())
		return
	}

	w.client.Post(w.endpoint).
		Send(string(b)).
		End()
}
