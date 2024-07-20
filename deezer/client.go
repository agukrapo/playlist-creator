package deezer

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

// Client represents a Deezer client.
type Client struct {
	httpClient doer
	apiURL     string

	tokenizer func(ctx context.Context) (string, cookieJar, error)

	arl string
}

func New(httpClient doer, arl string) *Client {
	out := &Client{
		httpClient: httpClient,
		apiURL:     "https://www.deezer.com/ajax/gw-light.php",
		arl:        arl,
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

func (c *Client) token(ctx context.Context) (string, cookieJar, error) {
	arl := newJar(&http.Cookie{Name: "arl", Value: c.arl})

	var out userResponse
	cookies, err := c.send(ctx, "", "deezer.getUserData", arl, nil, &out)
	if err != nil {
		return "", nil, err
	}

	if out.User.ID == 0 {
		return "", nil, errors.New("invalid arl cookie")
	}

	return out.CheckForm, cookies, nil
}

type searchResponse struct {
	Track struct {
		Data []struct {
			SongID string `json:"SNG_ID"`
			Artist string `json:"ART_NAME"`
			Title  string `json:"SNG_TITLE"`
			Album  string `json:"ALB_TITLE"`
		} `json:"data"`
	} `json:"TRACK"`
}

func (sr searchResponse) tracks() []playlists.Track {
	out := make([]playlists.Track, 0, len(sr.Track.Data))
	for _, t := range sr.Track.Data {
		if !validID(t.SongID) {
			continue
		}

		out = append(out, playlists.Track{
			ID:   t.SongID,
			Name: fmt.Sprintf("%s - %s <%s>", t.Artist, t.Title, t.Album),
		})
	}
	return out
}

func (c *Client) SearchTrack(ctx context.Context, query string) ([]playlists.Track, error) {
	token, cookies, err := c.tokenizer(ctx)
	if err != nil {
		return nil, err
	}

	in := map[string]string{"query": query}

	var out searchResponse
	if _, err := c.send(ctx, token, "deezer.pageSearch", cookies, in, &out); err != nil {
		return nil, err
	}

	return out.tracks(), nil
}

func (c *Client) CreatePlaylist(ctx context.Context, title string) (string, error) {
	token, cookies, err := c.tokenizer(ctx)
	if err != nil {
		return "", err
	}

	in := map[string]any{"title": title}

	var out json.Number
	if _, err := c.send(ctx, token, "playlist.create", cookies, in, &out); err != nil {
		return "", err
	}

	if !validID(out.String()) {
		return "", errors.New("failed to create playlist")
	}

	return out.String(), nil
}

func (c *Client) PopulatePlaylist(ctx context.Context, playlist string, tracks []string) error {
	token, cookies, err := c.tokenizer(ctx)
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
	if _, err := c.send(ctx, token, "playlist.addSongs", cookies, in, &out); err != nil {
		return err
	}

	if !out {
		return errors.New("failed to add tracks")
	}

	return nil
}

type envelope struct {
	Error   any             `json:"error"`
	Results json.RawMessage `json:"results"`
}

func (e envelope) asError() error {
	var out error

	switch t := e.Error.(type) {
	case map[string]any:
		for _, v := range t {
			out = errors.Join(out, uncapitalize(v))
		}
	case []any:
		for _, v := range t {
			out = errors.Join(out, uncapitalize(v))
		}
	default:
		if t != nil {
			out = uncapitalize(t)
		}
	}

	return out
}

func uncapitalize(v any) error {
	if v == nil {
		return nil
	}

	str := fmt.Sprint(v)
	if str == "" {
		return nil
	}

	return errors.New(strings.ToLower(string(str[0])) + str[1:])
}

type cookieJar map[string]*http.Cookie

func newJar(cookies ...*http.Cookie) cookieJar {
	out := make(cookieJar, len(cookies))
	for _, cookie := range cookies {
		out[cookie.Name] = cookie
	}
	return out
}

func (c *Client) send(ctx context.Context, token, method string, cookies cookieJar, in, out any) (cookieJar, error) {
	req, err := requests.New(c.apiURL).Post().JSON(in).Build(ctx)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Cache-Control", "max-age=0")

	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	vs := url.Values{}
	vs.Add("api_version", "1.0")
	vs.Add("api_token", token)
	vs.Add("method", method)
	req.URL.RawQuery = vs.Encode()

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if len(raw) == 0 {
		return nil, fmt.Errorf("%s: empty response", method)
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", res.Status, raw)
	}

	var env envelope
	if err = json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}

	if err := env.asError(); err != nil {
		return nil, err
	}

	if err := json.Unmarshal(env.Results, out); err != nil {
		return nil, err
	}

	return newJar(res.Cookies()...), nil
}

func validID(id string) bool {
	return id != "" && id != "0"
}
