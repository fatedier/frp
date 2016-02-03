package models

import (
	"container/list"
	"sync"

	"github.com/fatedier/frp/pkg/utils/conn"
	"github.com/fatedier/frp/pkg/utils/log"
)

const (
	Idle = iota
	Working
)

type ProxyServer struct {
	Name       string
	Passwd     string
	BindAddr   string
	ListenPort int64

	Status       int64
	Listener     *conn.Listener  // accept new connection from remote users
	CtlMsgChan   chan int64      // every time accept a new user conn, put "1" to the channel
	CliConnChan  chan *conn.Conn // get client conns from control goroutine
	UserConnList *list.List      // store user conns
	Mutex        sync.Mutex
}

func (p *ProxyServer) Init() {
	p.Status = Idle
	p.CtlMsgChan = make(chan int64)
	p.CliConnChan = make(chan *conn.Conn)
	p.UserConnList = list.New()
}

func (p *ProxyServer) Lock() {
	p.Mutex.Lock()
}

func (p *ProxyServer) Unlock() {
	p.Mutex.Unlock()
}

// start listening for user conns
func (p *ProxyServer) Start() (err error) {
	p.Listener, err = conn.Listen(p.BindAddr, p.ListenPort)
	if err != nil {
		return err
	}

	p.Status = Working

	// start a goroutine for listener
	go func() {
		for {
			// block
			c := p.Listener.GetConn()
			log.Debug("ProxyName [%s], get one new user conn [%s]", p.Name, c.GetRemoteAddr())

			// put to list
			p.Lock()
			if p.Status != Working {
				log.Debug("ProxyName [%s] is not working, new user conn close", p.Name)
				c.Close()
				p.Unlock()
				return
			}
			p.UserConnList.PushBack(c)
			p.Unlock()

			// put msg to control conn
			p.CtlMsgChan <- 1
		}
	}()

	// start another goroutine for join two conns from client and user
	go func() {
		for {
			cliConn := <-p.CliConnChan
			p.Lock()
			element := p.UserConnList.Front()

			var userConn *conn.Conn
			if element != nil {
				userConn = element.Value.(*conn.Conn)
				p.UserConnList.Remove(element)
			} else {
				cliConn.Close()
				continue
			}
			p.Unlock()

			// msg will transfer to another without modifying
			log.Debug("Join two conns, (l[%s] r[%s]) (l[%s] r[%s])", cliConn.GetLocalAddr(), cliConn.GetRemoteAddr(),
				userConn.GetLocalAddr(), userConn.GetRemoteAddr())
			go conn.Join(cliConn, userConn)
		}
	}()

	return nil
}

func (p *ProxyServer) Close() {
	p.Lock()
	p.Status = Idle
	p.CtlMsgChan = make(chan int64)
	p.CliConnChan = make(chan *conn.Conn)
	p.UserConnList = list.New()
	p.Unlock()
}

func (p *ProxyServer) WaitUserConn() (res int64) {
	res = <-p.CtlMsgChan
	return
}
