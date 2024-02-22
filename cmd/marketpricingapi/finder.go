package main

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"cloud.google.com/go/storage"

	"github.com/rs/xid"
)

type Finder struct {
	c            GCSClient
	baseLocation string
	bucket       string
	l            *zap.SugaredLogger
}

func NewFinder(c GCSClient, baseLocation, bucket string) Finder {
	return Finder{
		c:            c,
		baseLocation: baseLocation,
		bucket:       bucket,
		l:            zap.S(),
	}
}

type Result struct {
	FileName string
	Data     io.Reader
	closeFn  func() error
}

func (r *Result) Close() error {
	if r.closeFn != nil {
		return r.closeFn()
	}
	return nil
}

type LocalFile struct {
	Name string
	Time time.Time
}

const layout = "20060102_150405"

func getFileTime(name string) (int64, error) {
	baseName := filepath.Base(name)
	timeFormat := layout + ".log.gz"
	if len(baseName) < len(timeFormat) {
		return 0, fmt.Errorf("name not in a valid format")
	}
	t, err := time.Parse(timeFormat, baseName[len(baseName)-len(timeFormat):])
	if err != nil {
		return 0, fmt.Errorf("parse time: %w", err)
	}
	return t.Unix(), nil
}

func (f Finder) ListArchiveFile(baseLocation string) ([]LocalFile, error) {
	matches, err := filepath.Glob(filepath.Join(baseLocation, "*.log.gz"))
	if err != nil {
		return nil, fmt.Errorf("glob file: %w", err)
	}
	var res []LocalFile
	for _, v := range matches {
		fileTime, err := getFileTime(v)
		if err != nil {
			continue
		}
		res = append(res, LocalFile{
			Name: v,
			Time: time.Unix(fileTime, 0).UTC(),
		})
	}
	return res, nil
}

func (f Finder) GetLogRecord(logName string, reqID string) (string, error) {
	res, err := f.FindFile(logName, reqID)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = res.Close()
	}()
	f.l.Debugw("resolve find", "file", res.FileName)
	buffReader := bufio.NewReader(res.Data)
	for {
		line, err := buffReader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("cannot read line: %w", err)
		}
		if strings.Contains(line, reqID) {
			if jsonOpen := strings.Index(line, "{"); jsonOpen >= 0 {
				return line[jsonOpen:], nil
			}
			return line, nil
		}
	}
}

func (f Finder) tryCurrentFile(logName string, recordTime time.Time) (*Result, error) {
	currentFilePath := filepath.Join(f.baseLocation, logName+".log")
	/*currentFile, err := os.Stat(currentFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot stats current file %s: %w", currentFilePath, err)
	}
	if currentFile.ModTime().Before(recordTime) {*/
	fs, err := os.Open(currentFilePath)
	if err != nil {
		return nil, fmt.Errorf("cannot open current file %s: %w", currentFilePath, err)
	}
	return &Result{
		FileName: filepath.Base(currentFilePath),
		Data:     fs,
		closeFn: func() error {
			return fs.Close()
		},
	}, nil
}

func (f Finder) FindFile(logName, reqID string) (*Result, error) {
	id, err := xid.FromString(reqID)
	if err != nil {
		return nil, fmt.Errorf("id not valid: %w", err)
	}
	recordTime := id.Time().UTC()
	f.l.Debugw("item time", "time", recordTime)
	archiveFiles, err := f.ListArchiveFile(f.baseLocation)
	if err != nil {
		return nil, err
	}

	if len(archiveFiles) == 0 {
		return f.tryCurrentFile(logName, recordTime)
	}
	if recordTime.After(archiveFiles[0].Time) { // record time in archive file range
		for _, v := range archiveFiles {
			if v.Time.After(recordTime) { // found it
				filePath := v.Name
				fs, err := os.Open(filePath)
				if err != nil {
					return nil, fmt.Errorf("cannot open file %s: %w", filePath, err)
				}
				reader, err := gzip.NewReader(fs)
				if err != nil {
					_ = fs.Close()
					return nil, fmt.Errorf("open gzstream: %w", err)
				}
				return &Result{
					FileName: filepath.Base(v.Name),
					Data:     reader,
					closeFn: func() error {
						_ = reader.Close()
						return fs.Close()
					},
				}, nil
			}
		}
		return f.tryCurrentFile(logName, recordTime)
	}
	return f.findGCS(logName, recordTime)
}

func (f Finder) findGCS(logName string, recordTime time.Time) (*Result, error) {
	files, err := f.c.ListFiles(f.bucket, &storage.Query{
		MatchGlob: fmt.Sprintf("%s/%s/%s*.log.gz", logName, recordTime.Format("20060102"), logName),
	})
	if err != nil {
		return nil, err
	}
	for _, v := range files {
		if v.Time.After(recordTime) {
			of, err := f.c.OpenFile(f.bucket, v.Name)
			if err != nil {
				return nil, fmt.Errorf("cannot open remote file: %s/%s", f.bucket, v.Name)
			}
			reader, err := gzip.NewReader(of)
			if err != nil {
				_ = of.Close()
				return nil, fmt.Errorf("open remote gzstream: %w", err)
			}
			return &Result{
				FileName: v.Name,
				Data:     reader,
				closeFn: func() error {
					_ = reader.Close()
					return of.Close()
				},
			}, nil
		}
	}
	return nil, fmt.Errorf("not found on GCS")
}
