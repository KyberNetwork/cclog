package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/KyberNetwork/cclog/lib/client"
)

func main() {
	w2 := client.NewAsyncLogClient("test", "10.148.0.119:4560", func(err error) {
		fmt.Println("err", err)
	})
	w := io.MultiWriter(os.Stdout, w2)
	encoder := zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig())
	l := zap.New(zapcore.NewCore(encoder, zapcore.AddSync(w), zap.DebugLevel))
	defer func() {
		_ = l.Sync()
	}()
	s := l.Sugar()
	for range time.NewTicker(time.Second).C {
		s.Infow("this is first message", "key", "xxy")
	}
}
