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
	"time"
)

const defaultTimeout = 5 * time.Second

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	httpClient httpClient
	token      string
}

func New(token string) *Client {
	hc := &http.Client{
		Timeout: defaultTimeout,
	}

	return &Client{
		token:      token,
		httpClient: hc,
	}
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
}

type userResponse struct {
	ID string `json:"id"`
}

func (c *Client) Me(ctx context.Context) (string, error) {
	uri := "https://api.spotify.com/v1/me"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}

	c.headers(req)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", parseError(b)
	}

	var ur userResponse
	if err := json.Unmarshal(b, &ur); err != nil {
		return "", err
	}

	return ur.ID, nil
}

type searchResponse struct {
	Tracks struct {
		Items []struct {
			URI string `json:"uri"`
		} `json:"items"`
	} `json:"tracks"`
}

func (c *Client) SearchTrack(ctx context.Context, query string) (string, bool, error) {
	uri := "https://api.spotify.com/v1/search?type=track&q=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", false, err
	}

	c.headers(req)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", false, err
	}

	if res.StatusCode != http.StatusOK {
		return "", false, parseError(b)
	}

	var sr searchResponse
	if err := json.Unmarshal(b, &sr); err != nil {
		return "", false, err
	}

	if len(sr.Tracks.Items) == 0 {
		return "", false, nil
	}

	return sr.Tracks.Items[0].URI, true, nil
}

type playlistResponse struct {
	ID           string `json:"id"`
	ExternalUrls struct {
		Spotify string `json:"spotify"`
	} `json:"external_urls"`
}

func (c *Client) CreatePlaylist(ctx context.Context, userID, name string) (string, string, error) {
	uri := "https://api.spotify.com/v1/users/" + userID + "/playlists"
	body := strings.NewReader(fmt.Sprintf(`{"name":%q,"public":false}`, name))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, body)
	if err != nil {
		return "", "", err
	}

	c.headers(req)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return "", "", err
	}

	if res.StatusCode != http.StatusCreated {
		return "", "", parseError(b)
	}

	var pr playlistResponse
	if err := json.Unmarshal(b, &pr); err != nil {
		return "", "", err
	}

	return pr.ID, pr.ExternalUrls.Spotify, nil
}

func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []string) error {
	uri := "https://api.spotify.com/v1/playlists/" + playlistID + "/tracks"
	body := strings.NewReader(fmt.Sprintf(`{"uris":["%s"]}`, strings.Join(tracks, `","`)))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, body)
	if err != nil {
		return err
	}

	c.headers(req)

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		b, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		return parseError(b)
	}

	return err
}

type erroneousResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func parseError(b []byte) error {
	var er *erroneousResponse
	if err := json.Unmarshal(b, &er); err != nil {
		return err
	}

	return errors.New(er.Error.Message)
}
