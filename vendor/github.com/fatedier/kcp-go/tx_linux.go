// +build linux

package kcp

import (
	"net"
	"os"
	"sync/atomic"

	"github.com/pkg/errors"
	"golang.org/x/net/ipv4"
)

func (s *UDPSession) tx(txqueue []ipv4.Message) {
	// default version
	if s.xconn == nil || s.xconnWriteError != nil {
		s.defaultTx(txqueue)
		return
	}

	// x/net version
	nbytes := 0
	npkts := 0
	for len(txqueue) > 0 {
		if n, err := s.xconn.WriteBatch(txqueue, 0); err == nil {
			for k := range txqueue[:n] {
				nbytes += len(txqueue[k].Buffers[0])
				xmitBuf.Put(txqueue[k].Buffers[0])
			}
			npkts += n
			txqueue = txqueue[n:]
		} else {
			// compatibility issue:
			// for linux kernel<=2.6.32, support for sendmmsg is not available
			// an error of type os.SyscallError will be returned
			if operr, ok := err.(*net.OpError); ok {
				if se, ok := operr.Err.(*os.SyscallError); ok {
					if se.Syscall == "sendmmsg" {
						s.xconnWriteError = se
						s.defaultTx(txqueue)
						return
					}
				}
			}
			s.notifyWriteError(errors.WithStack(err))
			break
		}
	}

	atomic.AddUint64(&DefaultSnmp.OutPkts, uint64(npkts))
	atomic.AddUint64(&DefaultSnmp.OutBytes, uint64(nbytes))
}
