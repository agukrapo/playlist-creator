package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agukrapo/playlist-creator/internal/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Me(t *testing.T) {
	table := []struct {
		name           string
		responseStatus int
		responseBody   string
		expected       string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusOK,
			responseBody:   tests.ReadFile(t, "test-data/me_ok.json"),
			expected:       "username",
		},
		{
			name:           "error",
			responseStatus: http.StatusUnauthorized,
			responseBody:   tests.ReadFile(t, "test-data/me_error.json"),
			expectedError:  "Invalid access token",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/v1/me", req.URL.Path)
				assert.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				assert.Empty(t, tests.ReadBody(t, req))

				w.WriteHeader(test.responseStatus)
				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			err := client.Setup(context.Background())
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Equal(t, test.expected, client.userID)
		})
	}
}

func TestClient_SearchTrack(t *testing.T) {
	table := []struct {
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
			responseBody:   tests.ReadFile(t, "test-data/search_track_ok.json"),
			expectedURI:    "spotify:track:58W2OncAqstyVAumWdwTOz",
			expectedFound:  true,
		},
		{
			name:           "error",
			responseStatus: http.StatusBadRequest,
			responseBody:   tests.ReadFile(t, "test-data/search_track_error.json"),
			expectedError:  "No search query",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, "/v1/search", req.URL.Path)
				assert.Equal(t, "type=track&q=query", req.URL.RawQuery)
				assert.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				assert.Empty(t, tests.ReadBody(t, req))

				w.WriteHeader(test.responseStatus)
				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			uri, err := client.SearchTrack(context.Background(), "query")
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Equal(t, test.expectedURI, uri)
		})
	}
}

func TestClient_CreatePlaylist(t *testing.T) {
	table := []struct {
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
			responseBody:   tests.ReadFile(t, "test-data/create_playlist_ok.json"),
			expectedID:     "ujEWyhJniu4K7Kamfiki",
			expectedURL:    "https://open.spotify.com/playlist/ujEWyhJniu4K7Kamfiki",
		},
		{
			name:           "error",
			responseStatus: http.StatusForbidden,
			responseBody:   tests.ReadFile(t, "test-data/create_playlist_error.json"),
			expectedError:  "Insufficient client scope",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/v1/users/userID/playlists", req.URL.Path)
				assert.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				assert.Equal(t, `{"name":"playlistName","public":false}`, tests.ReadBody(t, req))

				w.WriteHeader(test.responseStatus)
				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
				userID:     "userID",
			}

			id, err := client.CreatePlaylist(context.Background(), "playlistName")
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Equal(t, test.expectedID, id)
		})
	}
}

func TestClient_AddTracksToPlaylist(t *testing.T) {
	table := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusCreated,
			responseBody:   tests.ReadFile(t, "test-data/add_tracks_to_playlist_ok.json"),
		},
		{
			name:           "error",
			responseStatus: http.StatusNotFound,
			responseBody:   tests.ReadFile(t, "test-data/add_tracks_to_playlist_error.json"),
			expectedError:  "Invalid playlist Id",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/v1/playlists/playlistID/tracks", req.URL.Path)
				assert.Equal(t, "Bearer oauth-token", req.Header.Get("Authorization"))
				assert.Equal(t, "application/json", req.Header.Get("Accept"))
				assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
				assert.Equal(t, `{"uris":["trackID"]}`, tests.ReadBody(t, req))

				w.WriteHeader(test.responseStatus)
				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			err := client.PopulatePlaylist(context.Background(), "playlistID", []string{"trackID"})
			require.Equal(t, test.expectedError, tests.AsString(err))
		})
	}
}
