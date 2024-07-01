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

func ReadCookie(t *testing.T, req *http.Request, name string) string {
	t.Helper()

	cookie, err := req.Cookie(name)
	require.NoError(t, err)

	return cookie.Value
}

func ReadFile(t *testing.T, path string) string {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)

	bytes, err := io.ReadAll(f)
	require.NoError(t, err)

	return string(bytes)
}

func AsString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
