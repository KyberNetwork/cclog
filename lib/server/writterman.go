package server

import (
	"path/filepath"
	"sync"

	"github.com/robfig/cron/v3"
)

type WriterMan struct {
	allWriter   map[string]*RotateLogWriter
	lock        sync.Mutex
	baseDir     string
	maxFileSize uint64
}

func NewWriterMan(baseDir string, maxFileSize uint64) *WriterMan {
	g := &WriterMan{
		allWriter:   make(map[string]*RotateLogWriter),
		baseDir:     baseDir,
		maxFileSize: maxFileSize,
	}
	c := cron.New()
	_, _ = c.AddFunc("0 0 * * *", g.dailyRotate)
	c.Start()
	return g
}
func (w *WriterMan) dailyRotate() {
	var aw []*RotateLogWriter
	w.lock.Lock()
	for _, o := range w.allWriter {
		aw = append(aw, o)
	}
	w.lock.Unlock()
	for _, o := range aw {
		o := o
		go func() {
			_ = o.Rotate()
		}()
	}
}
func (w *WriterMan) GetOrCreate(name string) *RotateLogWriter {
	w.lock.Lock()
	defer w.lock.Unlock()
	res, ok := w.allWriter[name]
	if !ok {
		res = NewRotateLogWriter(filepath.Join(w.baseDir, name), name+".log", w.maxFileSize)
		w.allWriter[name] = res
	}
	return res
}
