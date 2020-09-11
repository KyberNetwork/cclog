package client

import (
	"fmt"
	"net"
	"time"

	"cclog/lib/common"
)

type SendFailedFn = func(error)

type AsyncLogClient struct {
	remoteAddr   string
	streamClient net.Conn
	closeChan    chan struct{}
	buffer       chan []byte
	failedFn     SendFailedFn
	name         string
}

const (
	defaultBufferSize = 100
	backOffSeconds    = 1.0
)

func NewAsyncLogClient(name string, remoteAddr string, fn SendFailedFn) *AsyncLogClient {
	return NewAsyncLogClientWithBuffer(name, remoteAddr, fn, defaultBufferSize)
}
func NewAsyncLogClientWithBuffer(name string, remoteAddr string, fn SendFailedFn, bufferSize int) *AsyncLogClient {
	c := &AsyncLogClient{
		name:       name,
		remoteAddr: remoteAddr,
		buffer:     make(chan []byte, bufferSize),
		closeChan:  make(chan struct{}),
		failedFn:   fn,
	}
	go c.loop()
	return c
}

func (l *AsyncLogClient) Write(p []byte) (n int, err error) {
	select {
	case l.buffer <- p:
		break
	default:
		l.failedFn(fmt.Errorf("failed to append log"))
	}
	return len(p), nil
}

func (l *AsyncLogClient) Close() error {
	close(l.closeChan)
	return nil
}

func (l *AsyncLogClient) loop() {
	lastConnect := time.Now().Add(-2 * time.Second)
	write := func(data []byte) {
		var err error
		if l.streamClient == nil {
			secs := time.Since(lastConnect).Seconds()
			if secs < backOffSeconds {
				// skip due recent reconnect failed, we drop data as we can't hold
				return
			}
			lastConnect = time.Now()
			l.streamClient, err = net.Dial("tcp", l.remoteAddr)
			if err != nil {
				l.failedFn(fmt.Errorf("failed to connect, %w", err))
				return
			}
			err = common.WriteConnectRequest(l.streamClient, common.ConnectRequest{Name: l.name})
			if err != nil {
				l.failedFn(fmt.Errorf("write connect request failed, %w", err))
				return
			}
			resp, err := common.ReadConnectResponse(l.streamClient)
			if err != nil {
				l.failedFn(fmt.Errorf("read failed, %w", err))
				return
			}
			if !resp.Success {
				l.failedFn(fmt.Errorf("server return error, %s", resp.Status))
				_ = l.streamClient.Close()
				l.streamClient = nil
				return
			}
		}
		_, err = l.streamClient.Write(data)
		if err != nil {
			l.failedFn(fmt.Errorf("write failed, %w", err))
			_ = l.streamClient.Close()
			l.streamClient = nil
		}
	}

	for {
		select {
		case b := <-l.buffer:
			write(b)
		case <-l.closeChan:
			break
		}
	}
}
