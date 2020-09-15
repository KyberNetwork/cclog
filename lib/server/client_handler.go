package server

import (
	"net"
	"regexp"

	"go.uber.org/zap"

	"github.com/KyberNetwork/cclog/lib/common"
)

const (
	readBufferSize = 1 << 14
)

var (
	nameGrep = regexp.MustCompile(`^[0-9a-zA-Z-_]+$`)
)

type ClientHandler struct {
	conn net.Conn
	l    *zap.SugaredLogger
	wMan *WriterMan
}

func NewClientHandler(c net.Conn, wm *WriterMan) *ClientHandler {
	return &ClientHandler{
		conn: c,
		wMan: wm,
		l:    zap.S(),
	}
}

func (c *ClientHandler) Stop() {
	_ = c.conn.Close()
}

func (c *ClientHandler) Run() {
	defer func() {
		_ = c.conn.Close()
	}()
	req, err := common.ReadConnectRequest(c.conn)
	if err != nil {
		c.l.Errorw("read connect req failed", "err", err)
		return
	}
	res := common.ConnectResponse{
		Success: true,
		Status:  "OK",
	}
	match := nameGrep.MatchString(req.Name)
	if !match {
		res.Status = "name can only contain alpha char"
		res.Success = false
	}
	if err := common.WriteConnectResponse(c.conn, res); err != nil {
		c.l.Errorw("sent reply failed", "err", err)
		return
	}
	if !match {
		return
	}
	remote := c.conn.RemoteAddr()
	l := c.l.With("from", remote.String(), "name", req.Name)
	wLog := c.wMan.GetOrCreate(req.Name)
	buff := make([]byte, readBufferSize)
	for {
		n, err := c.conn.Read(buff)
		if err != nil {
			l.Errorw("read failed", "err", err)
			break
		}
		nw, err := wLog.Write(buff[:n])
		if err != nil {
			l.Errorw("write failed", "err", err)
			break
		}
		if nw != n {
			l.Errorw("short write", "nw", nw, "src_length", n)
			break
		}
	}
}
