package rtsp

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
)

const (
	interleavedPacketSize = 0x10000
	interleavedHeaderSize = 4
)

type TransportTCP struct {
	reader    *bufio.Reader
	onceError sync.Once
	err       chan error
}

func NewTransportTCP(reader *bufio.Reader) *TransportTCP {
	return &TransportTCP{
		reader: reader,
		err:    make(chan error, 1),
	}
}

func (t *TransportTCP) loop(wg *sync.WaitGroup, handler MediaHandler, onError func(error)) {
	defer wg.Done()

	buf := make([]byte, interleavedPacketSize)

	var (
		size, skip  int
		transportID int
	)

	for {
		if size == 0 {
			n, err := t.reader.Read(buf[skip:interleavedHeaderSize])
			if err != nil {
				if errors.Is(err, net.ErrClosed) {
					return
				}

				onError(fmt.Errorf("read interleved header: %w", err))
				return
			}
			skip += n

			if skip == interleavedHeaderSize {
				if buf[0] != '$' {
					onError(fmt.Errorf("invalid interleved header"))
					return
				}

				transportID = int(buf[1])
				size = int(binary.BigEndian.Uint16(buf[2:4]))
				skip = 0
			}

			continue
		}

		n, err := t.reader.Read(buf[skip:size])
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}

			onError(fmt.Errorf("read interleaved packet: %w", err))
			return
		}
		skip += n

		if skip == size {
			id := transportID >> 1
			packet := buf[:size]

			if (transportID & 1) == 0 {
				handler.OnRTP(id, packet)
			} else {
				handler.OnRTCP(id, packet)
			}

			size = 0
			skip = 0
		}
	}
}

// Setup prepares the transport and returns the transport parameters.
func (t *TransportTCP) Setup(mediaID int) (string, error) {
	rtpId := mediaID * 2
	rtcpId := rtpId + 1

	transport := fmt.Sprintf(
		"RTP/AVP/TCP;unicast;interleaved=%d-%d",
		rtpId,
		rtcpId,
	)

	return transport, nil
}

// Play starts receigin RTP/RTCP packets.
func (t *TransportTCP) Play(handler MediaHandler) {
	var wg sync.WaitGroup

	wg.Add(1)
	go t.loop(&wg, handler, t.onError)

	go func() {
		wg.Wait()
		t.onError(nil)
	}()
}

func (t *TransportTCP) Close() {}

func (t *TransportTCP) onError(err error) {
	t.onceError.Do(func() {
		t.err <- err
	})
}

func (t *TransportTCP) Err() <-chan error {
	return t.err
}
