package cclog

import (
	"net"

	"go.uber.org/zap"
)

type Server struct {
	wm       *WriterMan
	bindAddr string
	l        *zap.SugaredLogger
	listener net.Listener
}

func NewServer(bindAddr string, wm *WriterMan) *Server {
	return &Server{
		wm:       wm,
		bindAddr: bindAddr,
		l:        zap.S(),
	}
}

func (s *Server) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", s.bindAddr)
	if err != nil {
		s.l.Errorw("failed to bind address", "err", err)
		return err
	}
	for {
		c, err := s.listener.Accept()
		if err != nil {
			s.l.Errorw("accept failed", "err", err)
			return err
		}
		cc := NewClientHandler(c, s.wm)
		go cc.Run()
	}
}

func (s *Server) Shutdown() error {
	return s.listener.Close()
}
