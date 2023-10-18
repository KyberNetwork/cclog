package client

import (
	"fmt"
	"github.com/KyberNetwork/cclog/lib/agent"
	"github.com/pierrec/lz4/v3"
	"io"
	"log"
	"net"
	"time"

	"github.com/KyberNetwork/cclog/lib/common"
)

type SendFailedFn = func(error)

type AsyncLogClient struct {
	remoteAddr  string
	closeChan   chan struct{}
	logHolder   *agent.LogHolder
	failedFn    SendFailedFn
	name        string
	compression bool
}

const (
	backOffSeconds = 1.0
)

func NewAsyncLogClient(name string, remoteAddr string, fn SendFailedFn) *AsyncLogClient {
	return NewAsyncLogClientWithBuffer(name, remoteAddr, fn, true)
}
func NewAsyncLogClientWithBuffer(name string, remoteAddr string, fn SendFailedFn, compression bool) *AsyncLogClient {
	c := &AsyncLogClient{
		name:        name,
		remoteAddr:  remoteAddr,
		logHolder:   agent.NewLogHolder(),
		closeChan:   make(chan struct{}),
		failedFn:    fn,
		compression: compression,
	}
	go c.loop()
	return c
}

func (l *AsyncLogClient) Write(p []byte) (n int, err error) {
	l.logHolder.Write(p)
	return len(p), nil
}

func (l *AsyncLogClient) Close() error {
	close(l.closeChan)
	return nil
}

func (l *AsyncLogClient) sendBuffer(w io.Writer, data []byte) error {
	_, err := w.Write(data)
	if err != nil {
		return err
	}
	if lw, ok := w.(*lz4.Writer); ok {
		return lw.Flush()
	}
	return nil
}

func (l *AsyncLogClient) loop() {
	lastConnect := time.Now().Add(-2 * time.Second)
	var (
		streamClient net.Conn
		writer       io.Writer
	)
	write := func(data []byte) {
		var err error
		if streamClient == nil {
			secs := time.Since(lastConnect).Seconds()
			if secs < backOffSeconds {
				// skip due recent reconnect failed, we drop data as we can't hold
				return
			}
			lastConnect = time.Now()
			streamClient, err = net.Dial("tcp", l.remoteAddr)
			if err != nil {
				l.failedFn(fmt.Errorf("failed to connect, %w", err))
				return
			}
			err = common.WriteConnectRequest(streamClient, common.ConnectRequest{Name: l.name, Compression: l.compression})
			if err != nil {
				l.failedFn(fmt.Errorf("write connect request failed, %w", err))
				return
			}
			resp, err := common.ReadConnectResponse(streamClient)
			if err != nil {
				l.failedFn(fmt.Errorf("read failed, %w", err))
				return
			}
			if !resp.Success {
				l.failedFn(fmt.Errorf("server return error, %s", resp.Status))
				_ = streamClient.Close()
				streamClient = nil
				return
			}
			if l.compression {
				writer = lz4.NewWriter(streamClient)
			} else {
				writer = streamClient
			}
		}
		log.Println("send data", "size", len(data))
		err = l.sendBuffer(writer, data)
		if err != nil {
			l.failedFn(fmt.Errorf("write failed, %w", err))
			_ = streamClient.Close()
			streamClient = nil
			writer = nil
		}
	}
	tick := time.NewTicker(time.Millisecond * 500)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			if buffer, ok := l.logHolder.GetAndClear(); ok {
				write(buffer.Bytes())
				buffer.Reset()
				agent.BufferPool.Put(buffer)
			}
		case <-l.closeChan:
			break
		}
	}
}
