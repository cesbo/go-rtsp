package rtsp

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
)

type conn struct {
	mediaID  int
	rtpConn  *net.UDPConn
	rtcpConn *net.UDPConn
	lock     sync.Mutex
}

func (c *conn) loopRTP(wg *sync.WaitGroup, handler MediaHandler, onError func(error)) {
	defer wg.Done()

	buf := make([]byte, 0x10000)

	for {
		n, _, err := c.rtpConn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			c.lock.Lock()
			c.rtpConn.Close()
			c.rtpConn = nil
			c.lock.Unlock()

			onError(fmt.Errorf("read rtp: %w", err))

			return
		}

		handler.OnRTP(c.mediaID, buf[:n])
	}
}

func (c *conn) loopRTCP(wg *sync.WaitGroup, handler MediaHandler, onError func(error)) {
	defer wg.Done()

	buf := make([]byte, 0x800)

	for {
		n, _, err := c.rtcpConn.ReadFromUDP(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			c.lock.Lock()
			c.rtcpConn.Close()
			c.rtcpConn = nil
			c.lock.Unlock()

			onError(fmt.Errorf("read rtcp: %w", err))

			return
		}

		handler.OnRTCP(c.mediaID, buf[:n])
	}
}

func (c *conn) start(wg *sync.WaitGroup, handler MediaHandler, onError func(error)) {
	wg.Add(2)
	go c.loopRTP(wg, handler, onError)
	go c.loopRTCP(wg, handler, onError)
}

func (c *conn) close() {
	c.lock.Lock()
	defer c.lock.Unlock()

	if c.rtpConn != nil {
		c.rtpConn.Close()
		c.rtpConn = nil
	}

	if c.rtcpConn != nil {
		c.rtcpConn.Close()
		c.rtcpConn = nil
	}
}

type TransportUDP struct {
	connList  []*conn
	onceClose sync.Once
	onceError sync.Once
	err       chan error
}

func NewTransportUDP() *TransportUDP {
	return &TransportUDP{
		err: make(chan error, 1),
	}
}

// Setup prepares the transport and returns the transport parameters.
func (t *TransportUDP) Setup(mediaID int) (string, error) {
	a := &net.UDPAddr{
		IP: net.IPv4zero,
	}

	var (
		err               error
		rtpPort, rtcpPort int
		rtpConn, rtcpConn *net.UDPConn
	)

	const (
		minPort = 10000
		maxPort = 65000
	)

	for {
		// rtpPort random number in range 10000-65000
		rtpPort = minPort + (rand.Intn(maxPort - minPort))
		// rtpPort should be even
		rtpPort &^= 1

		rtcpPort = rtpPort + 1

		a.Port = rtpPort
		rtpConn, err = net.ListenUDP("udp", a)
		if err != nil {
			continue
		}

		a.Port = rtcpPort
		rtcpConn, err = net.ListenUDP("udp", a)
		if err != nil {
			rtpConn.Close()
			continue
		}

		break
	}

	c := &conn{
		mediaID:  mediaID,
		rtpConn:  rtpConn,
		rtcpConn: rtcpConn,
	}
	t.connList = append(t.connList, c)

	transport := fmt.Sprintf(
		"RTP/AVP;unicast;client_port=%d-%d",
		rtpPort,
		rtcpPort,
	)

	return transport, nil
}

// Play starts receigin RTP/RTCP packets.
func (t *TransportUDP) Play(handler MediaHandler) {
	var wg sync.WaitGroup

	for _, c := range t.connList {
		c.start(&wg, handler, t.onError)
	}

	go func() {
		wg.Wait()
		t.onError(nil)
	}()
}

func (t *TransportUDP) Close() {
	t.onceClose.Do(func() {
		for _, c := range t.connList {
			c.close()
		}
	})
}

func (t *TransportUDP) onError(err error) {
	t.onceError.Do(func() {
		t.err <- err
	})
}

func (t *TransportUDP) Err() <-chan error {
	return t.err
}
