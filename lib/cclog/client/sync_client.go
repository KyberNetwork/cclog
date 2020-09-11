package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	"cclog/lib/common"
)

type SyncLogClient struct {
	remoteAddr   string
	streamClient net.Conn
	name         string
	lock         sync.Mutex
	lastConnect  time.Time
}

func NewSyncLogClient(name string, remoteAddr string) *SyncLogClient {
	return NewSyncLogClientWithBuffer(name, remoteAddr)
}
func NewSyncLogClientWithBuffer(name string, remoteAddr string) *SyncLogClient {
	c := &SyncLogClient{
		name:        name,
		remoteAddr:  remoteAddr,
		lastConnect: time.Now().Add(-time.Minute),
	}
	return c
}

func (l *SyncLogClient) Write(p []byte) (n int, err error) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.streamClient == nil {
		secs := time.Since(l.lastConnect).Seconds()
		if secs < backOffSeconds {
			// skip due recent reconnect failed, we drop data as we can't hold
			return
		}
		l.lastConnect = time.Now()
		l.streamClient, err = net.Dial("tcp", l.remoteAddr)
		if err != nil {
			return 0, err
		}
		err = common.WriteConnectRequest(l.streamClient, common.ConnectRequest{Name: l.name})
		if err != nil {
			return 0, err
		}
		var resp common.ConnectResponse
		resp, err = common.ReadConnectResponse(l.streamClient)
		if err != nil {
			return 0, err
		}
		if !resp.Success {
			fmt.Println("server error", resp.Status)
			_ = l.streamClient.Close()
			l.streamClient = nil
			return
		}
	}
	n, err = l.streamClient.Write(p)
	if err != nil {
		fmt.Printf("write failed, %+v", err)
		_ = l.streamClient.Close()
		l.streamClient = nil
	}
	return
}

func (l *SyncLogClient) Close() error {
	if l.streamClient != nil {
		return l.streamClient.Close()
	}
	return nil
}
