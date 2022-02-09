package port

import (
	"fmt"
	"net"
	"strconv"
	"sync"

	"k8s.io/apimachinery/pkg/util/sets"
)

type Allocator struct {
	reserved sets.Int
	used     sets.Int
	mu       sync.Mutex
}

// NewAllocator return a port allocator for testing.
// Example: from: 10, to: 20, mod 4, index 1
// Reserved ports: 13, 17
func NewAllocator(from int, to int, mod int, index int) *Allocator {
	pa := &Allocator{
		reserved: sets.NewInt(),
		used:     sets.NewInt(),
	}

	for i := from; i <= to; i++ {
		if i%mod == index {
			pa.reserved.Insert(i)
		}
	}
	return pa
}

func (pa *Allocator) Get() int {
	return pa.GetByName("")
}

func (pa *Allocator) GetByName(portName string) int {
	var builder *nameBuilder
	if portName == "" {
		builder = &nameBuilder{}
	} else {
		var err error
		builder, err = unmarshalFromName(portName)
		if err != nil {
			fmt.Println(err, portName)
			return 0
		}
	}

	pa.mu.Lock()
	defer pa.mu.Unlock()

	for i := 0; i < 20; i++ {
		port := pa.getByRange(builder.rangePortFrom, builder.rangePortTo)
		if port == 0 {
			return 0
		}

		l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			// Maybe not controlled by us, mark it used.
			pa.used.Insert(port)
			continue
		}
		l.Close()

		udpAddr, err := net.ResolveUDPAddr("udp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			continue
		}
		udpConn, err := net.ListenUDP("udp", udpAddr)
		if err != nil {
			// Maybe not controlled by us, mark it used.
			pa.used.Insert(port)
			continue
		}
		udpConn.Close()

		pa.used.Insert(port)
		return port
	}
	return 0
}

func (pa *Allocator) getByRange(from, to int) int {
	if from <= 0 {
		port, _ := pa.reserved.PopAny()
		return port
	}

	// choose a random port between from - to
	ports := pa.reserved.UnsortedList()
	for _, port := range ports {
		if port >= from && port <= to {
			return port
		}
	}
	return 0
}

func (pa *Allocator) Release(port int) {
	if port <= 0 {
		return
	}

	pa.mu.Lock()
	defer pa.mu.Unlock()

	if pa.used.Has(port) {
		pa.used.Delete(port)
		pa.reserved.Insert(port)
	}
}
