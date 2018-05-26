package ports

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	MinPort                    = 1
	MaxPort                    = 65535
	MaxPortReservedDuration    = time.Duration(24) * time.Hour
	CleanReservedPortsInterval = time.Hour
)

var (
	ErrPortAlreadyUsed = errors.New("port already used")
	ErrPortNotAllowed  = errors.New("port not allowed")
	ErrPortUnAvailable = errors.New("port unavailable")
	ErrNoAvailablePort = errors.New("no available port")
)

type PortCtx struct {
	ProxyName  string
	Port       int
	Closed     bool
	UpdateTime time.Time
}

type PortManager struct {
	reservedPorts map[string]*PortCtx
	usedPorts     map[int]*PortCtx
	freePorts     map[int]struct{}

	bindAddr string
	netType  string
	mu       sync.Mutex
}

func NewPortManager(netType string, bindAddr string, allowPorts map[int]struct{}) *PortManager {
	pm := &PortManager{
		reservedPorts: make(map[string]*PortCtx),
		usedPorts:     make(map[int]*PortCtx),
		freePorts:     make(map[int]struct{}),
		bindAddr:      bindAddr,
		netType:       netType,
	}
	if len(allowPorts) > 0 {
		for port, _ := range allowPorts {
			pm.freePorts[port] = struct{}{}
		}
	} else {
		for i := MinPort; i <= MaxPort; i++ {
			pm.freePorts[i] = struct{}{}
		}
	}
	go pm.cleanReservedPortsWorker()
	return pm
}

func (pm *PortManager) Acquire(name string, port int) (realPort int, err error) {
	portCtx := &PortCtx{
		ProxyName:  name,
		Closed:     false,
		UpdateTime: time.Now(),
	}

	var ok bool

	pm.mu.Lock()
	defer func() {
		if err == nil {
			portCtx.Port = realPort
		}
		pm.mu.Unlock()
	}()

	// check reserved ports first
	if port == 0 {
		if ctx, ok := pm.reservedPorts[name]; ok {
			if pm.isPortAvailable(ctx.Port) {
				realPort = ctx.Port
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
				return
			}
		}
	}

	if port == 0 {
		// get random port
		count := 0
		maxTryTimes := 5
		for k, _ := range pm.freePorts {
			count++
			if count > maxTryTimes {
				break
			}
			if pm.isPortAvailable(k) {
				realPort = k
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
				break
			}
		}
		if realPort == 0 {
			err = ErrNoAvailablePort
		}
	} else {
		// specified port
		if _, ok = pm.freePorts[port]; ok {
			if pm.isPortAvailable(port) {
				realPort = port
				pm.usedPorts[realPort] = portCtx
				pm.reservedPorts[name] = portCtx
				delete(pm.freePorts, realPort)
			} else {
				err = ErrPortUnAvailable
			}
		} else {
			if _, ok = pm.usedPorts[port]; ok {
				err = ErrPortAlreadyUsed
			} else {
				err = ErrPortNotAllowed
			}
		}
	}
	return
}

func (pm *PortManager) isPortAvailable(port int) bool {
	if pm.netType == "udp" {
		addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", pm.bindAddr, port))
		if err != nil {
			return false
		}
		l, err := net.ListenUDP("udp", addr)
		if err != nil {
			return false
		}
		l.Close()
		return true
	} else {
		l, err := net.Listen(pm.netType, fmt.Sprintf("%s:%d", pm.bindAddr, port))
		if err != nil {
			return false
		}
		l.Close()
		return true
	}
}

func (pm *PortManager) Release(port int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if ctx, ok := pm.usedPorts[port]; ok {
		pm.freePorts[port] = struct{}{}
		delete(pm.usedPorts, port)
		ctx.Closed = true
		ctx.UpdateTime = time.Now()
	}
}

// Release reserved port if it isn't used in last 24 hours.
func (pm *PortManager) cleanReservedPortsWorker() {
	for {
		time.Sleep(CleanReservedPortsInterval)
		pm.mu.Lock()
		for name, ctx := range pm.reservedPorts {
			if ctx.Closed && time.Since(ctx.UpdateTime) > MaxPortReservedDuration {
				delete(pm.reservedPorts, name)
			}
		}
		pm.mu.Unlock()
	}
}
