package spotify

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/agukrapo/go-http-client/requests"
	"github.com/agukrapo/playlist-creator/playlists"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client represents a Spotify client.
type Client struct {
	httpClient doer
	baseURL    string
	token      string
	userID     string
}

// New creates a new Client.
func New(httpClient doer, token string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    "https://api.spotify.com",
		token:      token,
	}
}

func (c *Client) headers() map[string]string {
	return map[string]string{
		"Authorization": "Bearer " + c.token,
		"Accept":        "application/json",
		"Content-Type":  "application/json",
	}
}

type userResponse struct {
	ID string `json:"id"`
}

func (c *Client) Name() string {
	return "spotify"
}

func (c *Client) Setup(ctx context.Context) error {
	req, err := requests.New(c.baseURL + "/v1/me").Headers(c.headers()).Build(ctx)
	if err != nil {
		return err
	}

	res, err := send[userResponse](c.httpClient, req, http.StatusOK)
	if err != nil {
		return err
	}

	c.userID = res.ID

	return nil
}

type searchResponse struct {
	Tracks struct {
		Items []struct {
			URI string `json:"uri"`
		} `json:"items"`
	} `json:"tracks"`
}

// SearchTrack searches for the given query and retrieves the first match.
func (c *Client) SearchTrack(ctx context.Context, query string) (string, error) {
	u := c.baseURL + "/v1/search?type=track&q=" + url.QueryEscape(query)

	req, err := requests.New(u).Headers(c.headers()).Build(ctx)
	if err != nil {
		return "", err
	}

	res, err := send[searchResponse](c.httpClient, req, http.StatusOK)
	if err != nil {
		return "", err
	}

	if len(res.Tracks.Items) == 0 {
		return "", playlists.ErrTrackNotFound
	}

	return res.Tracks.Items[0].URI, nil
}

type playlistResponse struct {
	ID string `json:"id"`
}

// CreatePlaylist creates a named playlist for the given user.
func (c *Client) CreatePlaylist(ctx context.Context, name string) (string, error) {
	u := c.baseURL + "/v1/users/" + c.userID + "/playlists"
	body := strings.NewReader(fmt.Sprintf(`{"name":%q,"public":false}`, name))

	req, err := requests.New(u).Method(http.MethodPost).Body(body).Headers(c.headers()).Build(ctx)
	if err != nil {
		return "", err
	}

	res, err := send[playlistResponse](c.httpClient, req, http.StatusCreated)
	if err != nil {
		return "", err
	}

	return res.ID, nil
}

type playlistTrackResponse struct{}

// PopulatePlaylist adds the given tracks to the given playlist.
func (c *Client) PopulatePlaylist(ctx context.Context, playlistID string, tracks []string) error {
	u := c.baseURL + "/v1/playlists/" + playlistID + "/tracks"
	body := strings.NewReader(fmt.Sprintf(`{"uris":["%s"]}`, strings.Join(tracks, `","`)))

	req, err := requests.New(u).Method(http.MethodPost).Body(body).Headers(c.headers()).Build(ctx)
	if err != nil {
		return err
	}

	_, err = send[playlistTrackResponse](c.httpClient, req, http.StatusCreated)

	return err
}

type response interface {
	userResponse | searchResponse | playlistResponse | playlistTrackResponse
}

func send[t response](client doer, req *http.Request, expectedStatus int) (*t, error) {
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if expectedStatus != res.StatusCode {
		return nil, parseError(res.Body)
	}

	var out t
	return &out, json.NewDecoder(res.Body).Decode(&out)
}

func parseError(body io.Reader) error {
	var er struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.NewDecoder(body).Decode(&er); err != nil {
		return err
	}

	return errors.New(er.Error.Message)
}
