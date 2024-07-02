package deezer

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agukrapo/spotify-playlist-creator/internal/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_token(t *testing.T) {
	table := []struct {
		name          string
		responseBody  string
		expectedError string
		expectedToken string
	}{
		{
			name:          "ok",
			responseBody:  tests.ReadFile(t, "test-data/token_ok.json"),
			expectedToken: "VGXuOjpOD9H-P.DSbNSwjHn.0Ki4T3Nt",
		},
		{
			name:          "error",
			responseBody:  tests.ReadFile(t, "test-data/token_error.json"),
			expectedError: "invalid api token",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/?api_token=&api_version=1.0&method=deezer.getUserData", req.URL.String())
				assert.Equal(t, "null", tests.ReadBody(t, req))
				assert.Equal(t, "_ARL", tests.ReadCookie(t, req, "arl"))

				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := New(http.DefaultClient, "_ARL")
			client.apiURL = svr.URL

			token, err := client.token(context.Background())
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Equal(t, test.expectedToken, token)
		})
	}
}

func TestClient_searchTrack(t *testing.T) {
	table := []struct {
		name          string
		responseBody  string
		expectedError string
		expectedTrack string
	}{
		{
			name:          "ok",
			responseBody:  tests.ReadFile(t, "test-data/search_track_ok.json"),
			expectedTrack: "6623366",
		},
		{
			name:          "error",
			responseBody:  tests.ReadFile(t, "test-data/search_track_error.json"),
			expectedError: "track not found",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/?api_token=_TOKEN&api_version=1.0&method=deezer.pageSearch", req.URL.String())
				assert.Equal(t, `{"query":"_QUERY"}`, tests.ReadBody(t, req))
				assert.Equal(t, "_ARL", tests.ReadCookie(t, req, "arl"))

				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := New(http.DefaultClient, "_ARL")
			client.apiURL = svr.URL
			client.tokenizer = func(context.Context) (string, error) {
				return "_TOKEN", nil
			}

			track, err := client.SearchTrack(context.Background(), "_QUERY")
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Equal(t, test.expectedTrack, track)
		})
	}
}
