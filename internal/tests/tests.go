package tests

import (
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func ReadBody(t *testing.T, req *http.Request) string {
	t.Helper()

	bytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	return string(bytes)
}

func ReadFile(t *testing.T, path string) string {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)

	bytes, err := io.ReadAll(f)
	require.NoError(t, err)

	return string(bytes)
}
