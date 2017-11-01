package kcp

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func iclock() int32 {
	return int32((time.Now().UnixNano() / 1000000) & 0xffffffff)
}

type DelayPacket struct {
	_ptr  []byte
	_size int
	_ts   int32
}

func (p *DelayPacket) Init(size int, src []byte) {
	p._ptr = make([]byte, size)
	p._size = size
	copy(p._ptr, src[:size])
}

func (p *DelayPacket) ptr() []byte    { return p._ptr }
func (p *DelayPacket) size() int      { return p._size }
func (p *DelayPacket) ts() int32      { return p._ts }
func (p *DelayPacket) setts(ts int32) { p._ts = ts }

type DelayTunnel struct{ *list.List }
type Random *rand.Rand
type LatencySimulator struct {
	current                        int32
	lostrate, rttmin, rttmax, nmax int
	p12                            DelayTunnel
	p21                            DelayTunnel
	r12                            *rand.Rand
	r21                            *rand.Rand
}

// lostrate: 往返一周丢包率的百分比，默认 10%
// rttmin：rtt最小值，默认 60
// rttmax：rtt最大值，默认 125
//func (p *LatencySimulator)Init(int lostrate = 10, int rttmin = 60, int rttmax = 125, int nmax = 1000):
func (p *LatencySimulator) Init(lostrate, rttmin, rttmax, nmax int) {
	p.r12 = rand.New(rand.NewSource(9))
	p.r21 = rand.New(rand.NewSource(99))
	p.p12 = DelayTunnel{list.New()}
	p.p21 = DelayTunnel{list.New()}
	p.current = iclock()
	p.lostrate = lostrate / 2 // 上面数据是往返丢包率，单程除以2
	p.rttmin = rttmin / 2
	p.rttmax = rttmax / 2
	p.nmax = nmax
}

// 发送数据
// peer - 端点0/1，从0发送，从1接收；从1发送从0接收
func (p *LatencySimulator) send(peer int, data []byte, size int) int {
	rnd := 0
	if peer == 0 {
		rnd = p.r12.Intn(100)
	} else {
		rnd = p.r21.Intn(100)
	}
	//println("!!!!!!!!!!!!!!!!!!!!", rnd, p.lostrate, peer)
	if rnd < p.lostrate {
		return 0
	}
	pkt := &DelayPacket{}
	pkt.Init(size, data)
	p.current = iclock()
	delay := p.rttmin
	if p.rttmax > p.rttmin {
		delay += rand.Int() % (p.rttmax - p.rttmin)
	}
	pkt.setts(p.current + int32(delay))
	if peer == 0 {
		p.p12.PushBack(pkt)
	} else {
		p.p21.PushBack(pkt)
	}
	return 1
}

// 接收数据
func (p *LatencySimulator) recv(peer int, data []byte, maxsize int) int32 {
	var it *list.Element
	if peer == 0 {
		it = p.p21.Front()
		if p.p21.Len() == 0 {
			return -1
		}
	} else {
		it = p.p12.Front()
		if p.p12.Len() == 0 {
			return -1
		}
	}
	pkt := it.Value.(*DelayPacket)
	p.current = iclock()
	if p.current < pkt.ts() {
		return -2
	}
	if maxsize < pkt.size() {
		return -3
	}
	if peer == 0 {
		p.p21.Remove(it)
	} else {
		p.p12.Remove(it)
	}
	maxsize = pkt.size()
	copy(data, pkt.ptr()[:maxsize])
	return int32(maxsize)
}

//=====================================================================
//=====================================================================

// 模拟网络
var vnet *LatencySimulator

// 测试用例
func test(mode int) {
	// 创建模拟网络：丢包率10%，Rtt 60ms~125ms
	vnet = &LatencySimulator{}
	vnet.Init(10, 60, 125, 1000)

	// 创建两个端点的 kcp对象，第一个参数 conv是会话编号，同一个会话需要相同
	// 最后一个是 user参数，用来传递标识
	output1 := func(buf []byte, size int) {
		if vnet.send(0, buf, size) != 1 {
		}
	}
	output2 := func(buf []byte, size int) {
		if vnet.send(1, buf, size) != 1 {
		}
	}
	kcp1 := NewKCP(0x11223344, output1)
	kcp2 := NewKCP(0x11223344, output2)

	current := uint32(iclock())
	slap := current + 20
	index := 0
	next := 0
	var sumrtt uint32
	count := 0
	maxrtt := 0

	// 配置窗口大小：平均延迟200ms，每20ms发送一个包，
	// 而考虑到丢包重发，设置最大收发窗口为128
	kcp1.WndSize(128, 128)
	kcp2.WndSize(128, 128)

	// 判断测试用例的模式
	if mode == 0 {
		// 默认模式
		kcp1.NoDelay(0, 10, 0, 0)
		kcp2.NoDelay(0, 10, 0, 0)
	} else if mode == 1 {
		// 普通模式，关闭流控等
		kcp1.NoDelay(0, 10, 0, 1)
		kcp2.NoDelay(0, 10, 0, 1)
	} else {
		// 启动快速模式
		// 第二个参数 nodelay-启用以后若干常规加速将启动
		// 第三个参数 interval为内部处理时钟，默认设置为 10ms
		// 第四个参数 resend为快速重传指标，设置为2
		// 第五个参数 为是否禁用常规流控，这里禁止
		kcp1.NoDelay(1, 10, 2, 1)
		kcp2.NoDelay(1, 10, 2, 1)
	}

	buffer := make([]byte, 2000)
	var hr int32

	ts1 := iclock()

	for {
		time.Sleep(1 * time.Millisecond)
		current = uint32(iclock())
		kcp1.Update()
		kcp2.Update()

		// 每隔 20ms，kcp1发送数据
		for ; current >= slap; slap += 20 {
			buf := new(bytes.Buffer)
			binary.Write(buf, binary.LittleEndian, uint32(index))
			index++
			binary.Write(buf, binary.LittleEndian, uint32(current))
			// 发送上层协议包
			kcp1.Send(buf.Bytes())
			//println("now", iclock())
		}

		// 处理虚拟网络：检测是否有udp包从p1->p2
		for {
			hr = vnet.recv(1, buffer, 2000)
			if hr < 0 {
				break
			}
			// 如果 p2收到udp，则作为下层协议输入到kcp2
			kcp2.Input(buffer[:hr], true, false)
		}

		// 处理虚拟网络：检测是否有udp包从p2->p1
		for {
			hr = vnet.recv(0, buffer, 2000)
			if hr < 0 {
				break
			}
			// 如果 p1收到udp，则作为下层协议输入到kcp1
			kcp1.Input(buffer[:hr], true, false)
			//println("@@@@", hr, r)
		}

		// kcp2接收到任何包都返回回去
		for {
			hr = int32(kcp2.Recv(buffer[:10]))
			// 没有收到包就退出
			if hr < 0 {
				break
			}
			// 如果收到包就回射
			buf := bytes.NewReader(buffer)
			var sn uint32
			binary.Read(buf, binary.LittleEndian, &sn)
			kcp2.Send(buffer[:hr])
		}

		// kcp1收到kcp2的回射数据
		for {
			hr = int32(kcp1.Recv(buffer[:10]))
			buf := bytes.NewReader(buffer)
			// 没有收到包就退出
			if hr < 0 {
				break
			}
			var sn uint32
			var ts, rtt uint32
			binary.Read(buf, binary.LittleEndian, &sn)
			binary.Read(buf, binary.LittleEndian, &ts)
			rtt = uint32(current) - ts

			if sn != uint32(next) {
				// 如果收到的包不连续
				//for i:=0;i<8 ;i++ {
				//println("---", i, buffer[i])
				//}
				println("ERROR sn ", count, "<->", next, sn)
				return
			}

			next++
			sumrtt += rtt
			count++
			if rtt > uint32(maxrtt) {
				maxrtt = int(rtt)
			}

			//println("[RECV] mode=", mode, " sn=", sn, " rtt=", rtt)
		}

		if next > 100 {
			break
		}
	}

	ts1 = iclock() - ts1

	names := []string{"default", "normal", "fast"}
	fmt.Printf("%s mode result (%dms):\n", names[mode], ts1)
	fmt.Printf("avgrtt=%d maxrtt=%d\n", int(sumrtt/uint32(count)), maxrtt)
}

func TestNetwork(t *testing.T) {
	test(0) // 默认模式，类似 TCP：正常模式，无快速重传，常规流控
	test(1) // 普通模式，关闭流控等
	test(2) // 快速模式，所有开关都打开，且关闭流控
}

func BenchmarkFlush(b *testing.B) {
	kcp := NewKCP(1, func(buf []byte, size int) {})
	kcp.snd_buf = make([]segment, 32)
	for k := range kcp.snd_buf {
		kcp.snd_buf[k].xmit = 1
		kcp.snd_buf[k].resendts = currentMs() + 10000
	}
	b.ResetTimer()
	b.ReportAllocs()
	var mu sync.Mutex
	for i := 0; i < b.N; i++ {
		mu.Lock()
		kcp.flush(false)
		mu.Unlock()
	}
}
