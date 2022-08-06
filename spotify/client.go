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
)

type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

type Client struct {
	baseURL    string
	token      string
	httpClient httpClient
}

func New(token string, httpClient httpClient) *Client {
	return &Client{
		baseURL:    "https://api.spotify.com",
		token:      token,
		httpClient: httpClient,
	}
}

func (c *Client) headers(b *requests.Builder) *requests.Builder {
	b.Header("Authorization", "Bearer "+c.token)
	b.Header("Accept", "application/json")
	b.Header("Content-Type", "application/json")

	return b
}

type userResponse struct {
	ID string `json:"id"`
}

func (c *Client) Me(ctx context.Context) (string, error) {
	req, err := c.headers(requests.New(c.baseURL + "/v1/me")).Build(ctx)
	if err != nil {
		return "", err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	if res.StatusCode != http.StatusOK {
		return "", parseError(bytes)
	}

	var ur userResponse
	if err := json.Unmarshal(bytes, &ur); err != nil {
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
	uri := c.baseURL + "/v1/search?type=track&q=" + url.QueryEscape(query)

	req, err := c.headers(requests.New(uri)).Build(ctx)
	if err != nil {
		return "", false, err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", false, err
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", false, err
	}

	if res.StatusCode != http.StatusOK {
		return "", false, parseError(bytes)
	}

	var sr searchResponse
	if err := json.Unmarshal(bytes, &sr); err != nil {
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
	uri := c.baseURL + "/v1/users/" + userID + "/playlists"
	body := strings.NewReader(fmt.Sprintf(`{"name":%q,"public":false}`, name))

	req, err := c.headers(requests.New(uri).Method(http.MethodPost).Body(body)).Build(ctx)
	if err != nil {
		return "", "", err
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer res.Body.Close()

	bytes, err := io.ReadAll(res.Body)
	if err != nil {
		return "", "", err
	}

	if res.StatusCode != http.StatusCreated {
		return "", "", parseError(bytes)
	}

	var pr playlistResponse
	if err := json.Unmarshal(bytes, &pr); err != nil {
		return "", "", err
	}

	return pr.ID, pr.ExternalUrls.Spotify, nil
}

func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID string, tracks []string) error {
	uri := c.baseURL + "/v1/playlists/" + playlistID + "/tracks"
	body := strings.NewReader(fmt.Sprintf(`{"uris":["%s"]}`, strings.Join(tracks, `","`)))

	req, err := c.headers(requests.New(uri).Method(http.MethodPost).Body(body)).Build(ctx)
	if err != nil {
		return err
	}

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

func parseError(bytes []byte) error {
	var er *erroneousResponse
	if err := json.Unmarshal(bytes, &er); err != nil {
		return err
	}

	return errors.New(er.Error.Message)
}
