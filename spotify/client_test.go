package spotify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Me(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expected       string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusOK,
			responseBody:   readFile(t, "test-data/me_ok.json"),
			expected:       "username",
		},
		{
			name:           "error",
			responseStatus: http.StatusUnauthorized,
			responseBody:   readFile(t, "test-data/me_error.json"),
			expectedError:  "Invalid access token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/v1/me", req.URL.Path)
				require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				require.Equal(t, "application/json", req.Header.Get("Accept"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Empty(t, readBody(t, req))

				w.WriteHeader(tt.responseStatus)
				_, err := w.Write([]byte(tt.responseBody))
				require.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			id, err := client.Me(context.Background())
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Equal(t, err.Error(), tt.expectedError)
			}

			assert.Equal(t, tt.expected, id)
		})
	}
}

func TestClient_SearchTrack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedURI    string
		expectedFound  bool
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusOK,
			responseBody:   readFile(t, "test-data/search_track_ok.json"),
			expectedURI:    "spotify:track:58W2OncAqstyVAumWdwTOz",
			expectedFound:  true,
		},
		{
			name:           "error",
			responseStatus: http.StatusBadRequest,
			responseBody:   readFile(t, "test-data/search_track_error.json"),
			expectedError:  "No search query",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, "/v1/search", req.URL.Path)
				require.Equal(t, "type=track&q=query", req.URL.RawQuery)
				require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				require.Equal(t, "application/json", req.Header.Get("Accept"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Empty(t, readBody(t, req))

				w.WriteHeader(tt.responseStatus)
				_, err := w.Write([]byte(tt.responseBody))
				require.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}
			uri, found, err := client.SearchTrack(context.Background(), "query")
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.expectedError, err.Error())
			}

			assert.Equal(t, tt.expectedURI, uri)
			assert.Equal(t, tt.expectedFound, found)
		})
	}
}

func TestClient_CreatePlaylist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedID     string
		expectedURL    string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusCreated,
			responseBody:   readFile(t, "test-data/create_playlist_ok.json"),
			expectedID:     "ujEWyhJniu4K7Kamfiki",
			expectedURL:    "https://open.spotify.com/playlist/ujEWyhJniu4K7Kamfiki",
		},
		{
			name:           "error",
			responseStatus: http.StatusForbidden,
			responseBody:   readFile(t, "test-data/create_playlist_error.json"),
			expectedError:  "Insufficient client scope",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/v1/users/userID/playlists", req.URL.Path)
				require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				require.Equal(t, "application/json", req.Header.Get("Accept"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, `{"name":"playlistName","public":false}`, readBody(t, req))

				w.WriteHeader(tt.responseStatus)
				_, err := w.Write([]byte(tt.responseBody))
				require.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			id, url, err := client.CreatePlaylist(context.Background(), "userID", "playlistName")
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.expectedError, err.Error())
			}

			assert.Equal(t, tt.expectedID, id)
			assert.Equal(t, tt.expectedURL, url)
		})
	}
}

func TestClient_AddTracksToPlaylist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusCreated,
			responseBody:   readFile(t, "test-data/add_tracks_to_playlist_ok.json"),
		},
		{
			name:           "error",
			responseStatus: http.StatusNotFound,
			responseBody:   readFile(t, "test-data/add_tracks_to_playlist_error.json"),
			expectedError:  "Invalid playlist Id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodPost, req.Method)
				require.Equal(t, "/v1/playlists/playlistID/tracks", req.URL.Path)
				require.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				require.Equal(t, "application/json", req.Header.Get("Accept"))
				require.Equal(t, "application/json", req.Header.Get("Content-Type"))
				require.Equal(t, `{"uris":["trackID"]}`, readBody(t, req))

				w.WriteHeader(tt.responseStatus)
				_, err := w.Write([]byte(tt.responseBody))
				require.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			err := client.AddTracksToPlaylist(context.Background(), "playlistID", []string{"trackID"})
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.expectedError, err.Error())
			}
		})
	}
}

func readBody(t *testing.T, req *http.Request) string {
	t.Helper()

	bytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	return string(bytes)
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	f, err := os.Open(path)
	require.NoError(t, err)

	bytes, err := io.ReadAll(f)
	require.NoError(t, err)

	return string(bytes)
}
