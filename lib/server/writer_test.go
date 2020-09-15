package server

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNewRotateLogWriter(t *testing.T) {
	w := NewRotateLogWriter("./", "test.log", 300)
	for i := 0; i < 10; i++ {
		_, err := fmt.Fprintln(w, time.Now().String())
		require.NoError(t, err)
	}
	err := w.Close()
	require.NoError(t, err)
}
