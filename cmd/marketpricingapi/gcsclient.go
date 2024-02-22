package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"go.uber.org/zap"

	"cloud.google.com/go/storage"

	"google.golang.org/api/iterator"
)

type GCSClient struct {
	client *storage.Client
	l      *zap.SugaredLogger
}

func NewGCSClient(c *storage.Client) GCSClient {
	return GCSClient{client: c, l: zap.S()}
}

func (c GCSClient) ListFiles(bucketName string, q *storage.Query) ([]LocalFile, error) {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	res := make([]LocalFile, 0)
	it := c.client.Bucket(bucketName).Objects(ctx, q)
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("get attrs: %w", err)
		}
		ft, err := getFileTime(attrs.Name)
		if err != nil {
			c.l.Errorw("cannot parse file time", "name", attrs.Name, "err", err)
			continue
		}
		res = append(res, LocalFile{
			Name: attrs.Name,
			Time: time.Unix(ft, 0).UTC(),
		})
	}
	return res, nil
}

func (c GCSClient) OpenFile(bucket, object string) (io.ReadCloser, error) {
	ctx := context.Background()
	// ctx, cancel := context.WithTimeout(ctx, time.Second*120)
	// defer cancel()
	rc, err := c.client.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("Object(%q).NewReader: %w", object, err)
	}
	return rc, nil
}
