package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateClient(t *testing.T) {
	require.True(t, nameGrep.MatchString("abc12"))
	require.True(t, nameGrep.MatchString("abc12-"))
	require.False(t, nameGrep.MatchString("abc12?"))
	require.False(t, nameGrep.MatchString("abc12/"))
	require.False(t, nameGrep.MatchString("abc12\\"))
	require.False(t, nameGrep.MatchString("abc12."))
}
