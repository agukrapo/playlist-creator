package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"sync"

	"github.com/agukrapo/go-http-client/requests"
	"github.com/agukrapo/playlist-creator/playlists"
)

type doer interface {
	Do(*http.Request) (*http.Response, error)
}

// Client represents a Deezer client.
type Client struct {
	httpClient doer
	apiURL     string

	tokenizer func(ctx context.Context) (string, error)

	jar map[string]*http.Cookie
	mu  sync.RWMutex
}

func New(httpClient doer, arl string) *Client {
	out := &Client{
		httpClient: httpClient,
		apiURL:     "https://www.deezer.com/ajax/gw-light.php",
		jar:        map[string]*http.Cookie{"arl": {Name: "arl", Value: arl}},
	}

	out.tokenizer = out.token

	return out
}

func (c *Client) Name() string {
	return "deezer"
}

func (c *Client) Setup(context.Context) error {
	return nil
}

type userResponse struct {
	User struct {
		ID uint `json:"USER_ID"`
	}
	CheckForm string `json:"checkForm"`
}

func (c *Client) token(ctx context.Context) (string, error) {
	var out userResponse

	if err := c.send(ctx, "", "deezer.getUserData", nil, &out); err != nil {
		return "", err
	}

	if out.User.ID == 0 {
		return "", errors.New("invalid arl cookie")
	}

	return out.CheckForm, nil
}

type searchResponse struct {
	Track struct {
		Data []struct {
			SongID string `json:"SNG_ID"`
		} `json:"data"`
	} `json:"TRACK"`
}

func (c *Client) SearchTrack(ctx context.Context, query string) (string, error) {
	token, err := c.tokenizer(ctx)
	if err != nil {
		return "", err
	}

	in := map[string]string{"query": query}

	var out searchResponse
	if err := c.send(ctx, token, "deezer.pageSearch", in, &out); err != nil {
		return "", err
	}

	if len(out.Track.Data) == 0 || !validID(out.Track.Data[0].SongID) {
		return "", playlists.ErrTrackNotFound
	}

	return out.Track.Data[0].SongID, nil
}

func (c *Client) CreatePlaylist(ctx context.Context, title string) (string, error) {
	token, err := c.tokenizer(ctx)
	if err != nil {
		return "", err
	}

	in := map[string]any{"title": title}

	var out json.Number
	if err := c.send(ctx, token, "playlist.create", in, &out); err != nil {
		return "", err
	}

	if !validID(out.String()) {
		return "", errors.New("failed to create playlist")
	}

	return out.String(), nil
}

func (c *Client) PopulatePlaylist(ctx context.Context, playlist string, tracks []string) error {
	token, err := c.tokenizer(ctx)
	if err != nil {
		return err
	}

	songs := make([][]any, len(tracks))
	for i, t := range tracks {
		songs[i] = []any{t, i}
	}

	in := map[string]any{
		"playlist_id": playlist,
		"songs":       songs,
	}

	var out bool
	if err := c.send(ctx, token, "playlist.addSongs", in, &out); err != nil {
		return err
	}

	if !out {
		return errors.New("failed to add tracks")
	}

	return nil
}

type envelope struct {
	Error   any `json:"error"`
	Results any `json:"results"`
}

func (e envelope) asError() error {
	var out error

	switch t := e.Error.(type) {
	case map[string]any:
		for _, v := range t {
			out = errors.Join(out, fmt.Errorf("%v", v))
		}
	case []any:
		for _, v := range t {
			out = errors.Join(out, fmt.Errorf("%v", v))
		}
	default:
		if t != nil {
			out = fmt.Errorf("%v", t)
		}
	}

	return out
}

func (c *Client) send(ctx context.Context, token, method string, in, out any) error {
	req, err := requests.New(c.apiURL).Post().JSON(in).Build(ctx)
	if err != nil {
		return err
	}

	req.Header.Add("Cache-Control", "max-age=0")

	for _, cookie := range c.cookies() {
		req.AddCookie(cookie)
	}

	vs := url.Values{}
	vs.Add("api_version", "1.0")
	vs.Add("api_token", token)
	vs.Add("method", method)
	req.URL.RawQuery = vs.Encode()

	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: %s", res.Status, raw)
	}

	env := envelope{
		Results: out,
	}

	if err = json.Unmarshal(raw, &env); err != nil {
		return err
	}

	if err := env.asError(); err != nil {
		return err
	}

	c.saveCookies(res.Cookies())

	return nil
}

func (c *Client) cookies() map[string]*http.Cookie {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return maps.Clone(c.jar)
}

func (c *Client) saveCookies(cookies []*http.Cookie) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cookie := range cookies {
		c.jar[cookie.Name] = cookie
	}
}

func validID(id string) bool {
	return id != "" && id != "0"
}
