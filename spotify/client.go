package spotify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/agukrapo/go-http-client/requests"
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client represents a Spotify client.
type Client struct {
	baseURL    string
	token      string
	httpClient httpClient
}

// New creates a new Client.
func New(token string, httpClient httpClient) *Client {
	return &Client{
		baseURL:    "https://api.spotify.com",
		token:      token,
		httpClient: httpClient,
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

// Me retrieves the token's user id.
func (c *Client) Me(ctx context.Context) (string, error) {
	req, err := requests.New(c.baseURL + "/v1/me").Headers(c.headers()).Build(ctx)
	if err != nil {
		return "", err
	}

	res, err := sendAndParse[userResponse](c.httpClient, req, http.StatusOK)
	if err != nil {
		return "", err
	}

	return res.ID, nil
}

type searchResponse struct {
	Tracks struct {
		Items []struct {
			URI string `json:"uri"`
		} `json:"items"`
	} `json:"tracks"`
}

// SearchTrack searches for the given query and retrieves the first match.
func (c *Client) SearchTrack(ctx context.Context, query string) (string, bool, error) {
	url := c.baseURL + "/v1/search?type=track&q=" + url.QueryEscape(query)

	req, err := requests.New(url).Headers(c.headers()).Build(ctx)
	if err != nil {
		return "", false, err
	}

	res, err := sendAndParse[searchResponse](c.httpClient, req, http.StatusOK)
	if err != nil {
		return "", false, err
	}

	if len(res.Tracks.Items) == 0 {
		return "", false, nil
	}

	return res.Tracks.Items[0].URI, true, nil
}

type playlistResponse struct {
	ID           string `json:"id"`
	ExternalUrls struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

// CreatePlaylist creates a named playlist for the given user.
func (c *Client) CreatePlaylist(ctx context.Context, userID, name string) (string, string, error) {
	url := c.baseURL + "/v1/users/" + userID + "/playlists"
	body := strings.NewReader(fmt.Sprintf(`{"name":%q,"public":false}`, name))

	req, err := requests.New(url).Method(http.MethodPost).Body(body).Headers(c.headers()).Build(ctx)
	if err != nil {
		return "", "", err
	}

	res, err := sendAndParse[playlistResponse](c.httpClient, req, http.StatusCreated)
	if err != nil {
		return "", "", err
	}

	return res.ID, res.ExternalUrls.Spotify, nil
}

type playlistTrackResponse struct{}

// AddTracksToPlaylist adds the given tracks to the given playlist.
func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []string) error {
	url := c.baseURL + "/v1/playlists/" + playlistID + "/tracks"
	body := strings.NewReader(fmt.Sprintf(`{"uris":["%s"]}`, strings.Join(tracks, `","`)))

	req, err := requests.New(url).Method(http.MethodPost).Body(body).Headers(c.headers()).Build(ctx)
	if err != nil {
		return err
	}

	if _, err := sendAndParse[playlistTrackResponse](c.httpClient, req, http.StatusCreated); err != nil {
		return err
	}

	return nil
}

type response interface {
	userResponse | searchResponse | playlistResponse | playlistTrackResponse
}

func sendAndParse[T response](client httpClient, req *http.Request, spectedStatus int) (T, error) {
	var out T

	res, err := client.Do(req)
	if err != nil {
		return out, err
	}
	defer res.Body.Close()

	if res.StatusCode != spectedStatus {
		return out, parseError(res.Body)
	}

	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return out, err
	}

	return out, nil
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

	return fmt.Errorf("spotify: %s", er.Error.Message)
}
