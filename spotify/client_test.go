package spotify

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agukrapo/spotify-playlist-creator/internal/tests"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Me(t *testing.T) {
	suite := []struct {
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
			expectedError:  "spotify: Invalid access token",
		},
	}
	for _, test := range suite {
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

			id, err := client.Me(context.Background())
			if test.expectedError != "" {
				require.Equal(t, test.expectedError, err.Error())

				return
			}

			assert.Equal(t, test.expected, id)
		})
	}
}

func TestClient_SearchTrack(t *testing.T) {
	suite := []struct {
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
			expectedError:  "spotify: No search query",
		},
	}
	for _, test := range suite {
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

			uri, found, err := client.SearchTrack(context.Background(), "query")
			if test.expectedError == "" {
				require.NoError(t, err)

				return
			}

			require.Equal(t, test.expectedError, err.Error())
			assert.Equal(t, test.expectedURI, uri)
			assert.Equal(t, test.expectedFound, found)
		})
	}
}

func TestClient_CreatePlaylist(t *testing.T) {
	suite := []struct {
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
			expectedError:  "spotify: Insufficient client scope",
		},
	}
	for _, test := range suite {
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
			}

			id, url, err := client.CreatePlaylist(context.Background(), "userID", "playlistName")
			if test.expectedError != "" {
				require.Equal(t, test.expectedError, err.Error())

				return
			}

			assert.Equal(t, test.expectedID, id)
			assert.Equal(t, test.expectedURL, url)
		})
	}
}

func TestClient_AddTracksToPlaylist(t *testing.T) {
	suite := []struct {
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
			expectedError:  "spotify: Invalid playlist Id",
		},
	}
	for _, test := range suite {
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

			err := client.AddTracksToPlaylist(context.Background(), "playlistID", []string{"trackID"})
			if test.expectedError != "" {
				require.Equal(t, test.expectedError, err.Error())

				return
			}
		})
	}
}
