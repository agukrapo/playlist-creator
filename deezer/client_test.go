package deezer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agukrapo/playlist-creator/internal/logs"
	"github.com/agukrapo/playlist-creator/internal/tests"
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

			client := New(http.DefaultClient, "_ARL", logs.New(nil))
			client.apiURL = svr.URL

			token, cookies, err := client.token(context.Background())
			require.Equal(t, test.expectedError, tests.AsString(err))

			assert.Len(t, cookies, 0)
			assert.Equal(t, test.expectedToken, token)
		})
	}
}

func TestClient_SearchTrack(t *testing.T) {
	table := []struct {
		name                  string
		responseBody          string
		expectedMatchesLength int
		expectedID            string
		expectedName          string
	}{
		{
			name:                  "ok",
			responseBody:          tests.ReadFile(t, "test-data/search_track_ok.json"),
			expectedMatchesLength: 1,
			expectedID:            "6623366",
			expectedName:          "Porno For Pyros - Tahitian Moon <Good God's Urge>",
		},
		{
			name:         "error",
			responseBody: tests.ReadFile(t, "test-data/search_track_error.json"),
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

			client := New(http.DefaultClient, "_ARL", logs.New(nil))
			client.apiURL = svr.URL
			client.tokenizer = func(context.Context) (string, cookieJar, error) {
				return "_TOKEN", newJar(&http.Cookie{Name: "arl", Value: "_ARL"}), nil
			}

			matches, err := client.SearchTracks(context.Background(), "_QUERY")
			require.NoError(t, err)

			assert.Len(t, matches, test.expectedMatchesLength)

			if test.expectedMatchesLength == 0 {
				return
			}

			track := matches[0]
			assert.Equal(t, test.expectedID, track.ID)
			assert.Equal(t, test.expectedName, track.Name)
		})
	}
}

func TestClient_PopulatePlaylist(t *testing.T) {
	table := []struct {
		name          string
		responseBody  string
		expectedError string
	}{
		{
			name:         "ok",
			responseBody: tests.ReadFile(t, "test-data/populate_playlist_ok.json"),
		},
		{
			name:          "error",
			responseBody:  tests.ReadFile(t, "test-data/populate_playlist_error.json"),
			expectedError: "this song already exists in this playlist",
		},
	}
	for _, test := range table {
		t.Run(test.name, func(t *testing.T) {
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				assert.Equal(t, http.MethodPost, req.Method)
				assert.Equal(t, "/?api_token=_TOKEN&api_version=1.0&method=playlist.addSongs", req.URL.String())
				assert.Equal(t, `{"playlist_id":"_PLAYLIST_ID","songs":[["_TRACK_A",0]]}`, tests.ReadBody(t, req))
				assert.Equal(t, "_ARL", tests.ReadCookie(t, req, "arl"))

				_, err := w.Write([]byte(test.responseBody))
				assert.NoError(t, err)
			}))
			defer svr.Close()

			client := New(http.DefaultClient, "_ARL", logs.New(nil))
			client.apiURL = svr.URL
			client.tokenizer = func(context.Context) (string, cookieJar, error) {
				return "_TOKEN", newJar(&http.Cookie{Name: "arl", Value: "_ARL"}), nil
			}

			err := client.PopulatePlaylist(context.Background(), "_PLAYLIST_ID", []string{"_TRACK_A"})
			require.Equal(t, test.expectedError, tests.AsString(err))
		})
	}
}

func Test_uncapitalize(t *testing.T) {
	table := []struct {
		v        any
		expected error
	}{
		{
			v:        nil,
			expected: nil,
		},
		{
			v:        "",
			expected: nil,
		},
		{
			v:        "A",
			expected: errors.New("a"),
		},
		{
			v:        "AB",
			expected: errors.New("aB"),
		},
	}
	for _, test := range table {
		t.Run(fmt.Sprintf("%v->%v", test.v, test.expected), func(t *testing.T) {
			assert.Equal(t, test.expected, uncapitalize(test.v))
		})
	}
}
