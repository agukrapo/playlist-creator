package spotify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Me(t *testing.T) {
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
			responseBody: `{
				"display_name": "username",
				"external_urls": {
				  "spotify": "https://open.spotify.com/user/username"
				},
				"followers": {
				  "href": null,
				  "total": 0
				},
				"href": "https://api.spotify.com/v1/users/username",
				"id": "username",
				"images": [],
				"type": "user",
				"uri": "spotify:user:username"
			  }`,
			expected: "username",
		},
		{
			name:           "error",
			responseStatus: http.StatusUnauthorized,
			responseBody: `{
				"error": {
				  "status": 401,
				  "message": "Invalid access token"
				}
			  }`,
			expectedError: "Invalid access token",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			c := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			id, err := c.Me(context.Background())
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
			responseBody: `{
				"tracks": {
				  "href": "https://api.spotify.com/v1/search?query=query",
				  "items": [
					{
					  "album": {
						"album_type": "compilation",
						"artists": [
						  {
							"external_urls": {
							  "spotify": "https://open.spotify.com/artist/3RGLhK1IP9jnYFH4BRFJBS"
							},
							"href": "https://api.spotify.com/v1/artists/3RGLhK1IP9jnYFH4BRFJBS",
							"id": "3RGLhK1IP9jnYFH4BRFJBS",
							"name": "The Clash",
							"type": "artist",
							"uri": "spotify:artist:3RGLhK1IP9jnYFH4BRFJBS"
						  }
						],
						"external_urls": {
						  "spotify": "https://open.spotify.com/album/1IURjc5W1J7vtgKgaMe3BG"
						},
						"href": "https://api.spotify.com/v1/albums/1IURjc5W1J7vtgKgaMe3BG",
						"id": "1IURjc5W1J7vtgKgaMe3BG",
						"images": [
						  {
							"height": 640,
							"url": "https://i.scdn.co/image/ab67616d0000b273ef80d72dd413b7b22e81e743",
							"width": 640
						  },
						  {
							"height": 300,
							"url": "https://i.scdn.co/image/ab67616d00001e02ef80d72dd413b7b22e81e743",
							"width": 300
						  },
						  {
							"height": 64,
							"url": "https://i.scdn.co/image/ab67616d00004851ef80d72dd413b7b22e81e743",
							"width": 64
						  }
						],
						"name": "Super Black Market Clash",
						"release_date": "1993-10-26",
						"release_date_precision": "day",
						"total_tracks": 21,
						"type": "album",
						"uri": "spotify:album:1IURjc5W1J7vtgKgaMe3BG"
					  },
					  "artists": [
						{
						  "external_urls": {
							"spotify": "https://open.spotify.com/artist/3RGLhK1IP9jnYFH4BRFJBS"
						  },
						  "href": "https://api.spotify.com/v1/artists/3RGLhK1IP9jnYFH4BRFJBS",
						  "id": "3RGLhK1IP9jnYFH4BRFJBS",
						  "name": "The Clash",
						  "type": "artist",
						  "uri": "spotify:artist:3RGLhK1IP9jnYFH4BRFJBS"
						}
					  ],
					  "disc_number": 1,
					  "duration_ms": 265733,
					  "explicit": false,
					  "external_ids": {
						"isrc": "GBBBN0009372"
					  },
					  "external_urls": {
						"spotify": "https://open.spotify.com/track/58W2OncAqstyVAumWdwTOz"
					  },
					  "href": "https://api.spotify.com/v1/tracks/58W2OncAqstyVAumWdwTOz",
					  "id": "58W2OncAqstyVAumWdwTOz",
					  "is_local": false,
					  "name": "Mustapha Dance",
					  "popularity": 17,
					  "preview_url": "https://p.scdn.co/mp3-preview/55b5579eb0c839903c12ed9204dcec774adac601?cid=774b29d4f13844c495f206cafdad9c86",
					  "track_number": 21,
					  "type": "track",
					  "uri": "spotify:track:58W2OncAqstyVAumWdwTOz"
					}
				  ],
				  "limit": 20,
				  "next": null,
				  "offset": 0,
				  "previous": null,
				  "total": 1
				}
			  }`,
			expectedURI:   "spotify:track:58W2OncAqstyVAumWdwTOz",
			expectedFound: true,
		},
		{
			name:           "error",
			responseStatus: http.StatusBadRequest,
			responseBody: `{
				"error": {
				  "status": 400,
				  "message": "No search query"
				}
			  }`,
			expectedError: "No search query",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			c := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}
			uri, found, err := c.SearchTrack(context.Background(), "query")
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
			responseBody: `{
				"collaborative": false,
				"description": "playlistName description",
				"external_urls": {
				  "spotify": "https://open.spotify.com/playlist/ujEWyhJniu4K7Kamfiki"
				},
				"followers": {
				  "href": null,
				  "total": 0
				},
				"href": "https://api.spotify.com/v1/playlists/ujEWyhJniu4K7Kamfiki",
				"id": "ujEWyhJniu4K7Kamfiki",
				"images": [],
				"name": "playlistName",
				"owner": {
				  "display_name": "userID",
				  "external_urls": {
					"spotify": "https://open.spotify.com/user/userID"
				  },
				  "href": "https://api.spotify.com/v1/users/userID",
				  "id": "userID",
				  "type": "user",
				  "uri": "spotify:user:userID"
				},
				"primary_color": null,
				"public": false,
				"snapshot_id": "MSw0ZDQ3OTNjZGJmOWU0MzdhYTNkYjkyYjJkNzIxMDc0N2EyYTAyMTBl",
				"tracks": {
				  "href": "https://api.spotify.com/v1/playlists/ujEWyhJniu4K7Kamfiki/tracks",
				  "items": [],
				  "limit": 100,
				  "next": null,
				  "offset": 0,
				  "previous": null,
				  "total": 0
				},
				"type": "playlist",
				"uri": "spotify:playlist:ujEWyhJniu4K7Kamfiki"
			  }`,
			expectedID:  "ujEWyhJniu4K7Kamfiki",
			expectedURL: "https://open.spotify.com/playlist/ujEWyhJniu4K7Kamfiki",
		},
		{
			name:           "error",
			responseStatus: http.StatusForbidden,
			responseBody: `{
				"error": {
				  "status": 403,
				  "message": "Insufficient client scope"
				}
			  }`,
			expectedError: "Insufficient client scope",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			c := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			id, url, err := c.CreatePlaylist(context.Background(), "userID", "playlistName")
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
	tests := []struct {
		name           string
		responseStatus int
		responseBody   string
		expectedError  string
	}{
		{
			name:           "ok",
			responseStatus: http.StatusCreated,
			responseBody: `{
				"snapshot_id": "Yoea4z7kXXFLj7rzik3AYcCTPiVvkHRo5aEKVT7C"
			  }`,
		},
		{
			name:           "error",
			responseStatus: http.StatusNotFound,
			responseBody: `{
				"error": {
				  "status": 404,
				  "message": "Invalid playlist Id"
				}
			  }`,
			expectedError: "Invalid playlist Id",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			c := &Client{
				baseURL:    svr.URL,
				token:      "oauth-token",
				httpClient: http.DefaultClient,
			}

			err := c.AddTracksToPlaylist(context.Background(), "playlistID", []string{"trackID"})
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.Equal(t, tt.expectedError, err.Error())
			}
		})
	}
}

func readBody(t *testing.T, req *http.Request) string {
	bytes, err := io.ReadAll(req.Body)
	require.NoError(t, err)

	return string(bytes)
}
