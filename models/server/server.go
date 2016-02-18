package server

import (
	"container/list"
	"sync"

	"github.com/fatedier/frp/models/consts"
	"github.com/fatedier/frp/utils/conn"
	"github.com/fatedier/frp/utils/log"
)

type ProxyServer struct {
	Name       string
	Passwd     string
	BindAddr   string
	ListenPort int64

	Status        int64
	Listener      *conn.Listener  // accept new connection from remote users
	CtlMsgChan    chan int64      // every time accept a new user conn, put "1" to the channel
	StopBlockChan chan int64      // put any number to the channel, if you want to stop wait user conn
	CliConnChan   chan *conn.Conn // get client conns from control goroutine
	UserConnList  *list.List      // store user conns
	Mutex         sync.Mutex
}

func (p *ProxyServer) Init() {
	p.Status = consts.Idle
	p.CtlMsgChan = make(chan int64)
	p.StopBlockChan = make(chan int64)
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

	p.Status = consts.Working

	// start a goroutine for listener
	go func() {
		for {
			// block
			c := p.Listener.GetConn()
			log.Debug("ProxyName [%s], get one new user conn [%s]", p.Name, c.GetRemoteAddr())

			// put to list
			p.Lock()
			if p.Status != consts.Working {
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
				p.Unlock()
				continue
			}
			p.Unlock()

			// msg will transfer to another without modifying
			// l means local, r means remote
			log.Debug("Join two conns, (l[%s] r[%s]) (l[%s] r[%s])", cliConn.GetLocalAddr(), cliConn.GetRemoteAddr(),
				userConn.GetLocalAddr(), userConn.GetRemoteAddr())
			go conn.Join(cliConn, userConn)
		}
	}()

	return nil
}

func (p *ProxyServer) Close() {
	p.Lock()
	p.Status = consts.Idle
	p.CtlMsgChan = make(chan int64)
	p.CliConnChan = make(chan *conn.Conn)
	p.UserConnList = list.New()
	p.Unlock()
}

func (p *ProxyServer) WaitUserConn() (res int64, isStop bool) {
	select {
	case res = <-p.CtlMsgChan:
		return res, false
	case <-p.StopBlockChan:
		return 0, true
	}
}

func (p *ProxyServer) StopWaitUserConn() {
	p.StopBlockChan <- 1
}
