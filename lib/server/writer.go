package server

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type RotateLogWriter struct {
	currentFile     *os.File
	currentWriter   *bufio.Writer
	currentFileName string
	lock            sync.Mutex
	baseDir         string
	name            string
	maxSize         uint64
	currentWrite    uint64
}

func NewRotateLogWriter(baseDir string, name string, maxSize uint64) *RotateLogWriter {
	_ = os.MkdirAll(baseDir, 0755) // make sure dir is exists
	return &RotateLogWriter{
		baseDir:      baseDir,
		name:         name,
		maxSize:      maxSize,
		currentWrite: 0,
	}
}

func (r *RotateLogWriter) createOrOpenFile() (*os.File, string, uint64, error) {
	currentFileName := path.Join(r.baseDir, r.name)
	currentFile, err := os.OpenFile(currentFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, "", 0, err
	}
	ss, err := currentFile.Stat()
	if err != nil {
		return nil, "", 0, err
	}
	currentWrite := uint64(ss.Size())
	return currentFile, currentFileName, currentWrite, nil
}

func (r *RotateLogWriter) Write(p []byte) (n int, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if r.currentFile == nil {
		r.currentFile, r.currentFileName, r.currentWrite, err = r.createOrOpenFile()
		if err != nil {
			return 0, err
		}
		if r.currentWriter != nil {
			r.currentWriter.Reset(r.currentFile)
		} else {
			r.currentWriter = bufio.NewWriterSize(r.currentFile, 16*1024)
		}
	}
	n, err = r.currentWriter.Write(p)
	if n < 0 {
		panic("bytes written negative")
	}
	r.currentWrite += uint64(n)
	if r.currentWrite >= r.maxSize {
		err = r.rotate()
		if err != nil {
			err = fmt.Errorf("dailyRotate failed %w", err)
		}
	}
	return
}

func (r *RotateLogWriter) rotate() error {
	_ = r.currentWriter.Flush()
	defer func() {
		r.currentFileName = ""
	}()
	ext := filepath.Ext(r.currentFileName)
	var backupName string
	err := r.close()
	if err != nil {
		return err
	}
	for {
		backupName = r.currentFileName[:len(r.currentFileName)-len(ext)] + "-" +
			time.Now().Format("20060102_150405") + ext
		if bs, err := os.Stat(backupName); err == nil && bs.Size() > 0 {
			time.Sleep(time.Second)
			continue
		}
		break
	}

	err = os.Rename(r.currentFileName, backupName)
	if err != nil {
		return err
	}
	return nil
}

// have to call from func that keep lock object
func (r *RotateLogWriter) close() error {
	defer func() {
		r.currentWrite = 0
		r.currentFile = nil
	}()
	if r.currentFile != nil {
		return r.currentFile.Close()
	}
	return nil
}

func (r *RotateLogWriter) Rotate() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.rotate()
}

func (r *RotateLogWriter) Close() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.close()
}
