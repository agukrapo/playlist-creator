package deezer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/agukrapo/go-http-client/requests"
	"github.com/agukrapo/playlist-creator/internal/logs"
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

	log *logs.Logger
}

func New(httpClient doer, arl string, log *logs.Logger) *Client {
	out := &Client{
		httpClient: httpClient,
		apiURL:     "https://www.deezer.com/ajax/gw-light.php",
		arl:        arl,
		log:        log,
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

func (c *Client) token(ctx context.Context) (token string, cookies cookieJar, err error) {
	tr := c.log.Trace("deezer.token").Begins()
	defer func() { tr.Ends(err, logs.Var("token", token), logs.Var("cookies", cookies)) }()

	arl := newJar(&http.Cookie{Name: "arl", Value: c.arl})

	var out userResponse
	cookies, err = c.send(ctx, tr, "", "deezer.getUserData", arl, nil, &out)
	if err != nil {
		return "", nil, err
	}

	if out.User.ID == 0 {
		return "", nil, errors.New("invalid arl cookie")
	}

	return out.CheckForm, cookies, nil
}

type duration string

func (d duration) String() string {
	v, err := strconv.Atoi(string(d))
	if err != nil {
		return ""
	}

	dur, err := time.ParseDuration(fmt.Sprintf("%ds", v))
	if err != nil {
		return ""
	}

	return time.Unix(0, 0).UTC().Add(dur).Format("04:05")
}

type album struct {
	ID           string `json:"ALB_ID"`
	Title        string `json:"ALB_TITLE"`
	Date         string `json:"ORIGINAL_RELEASE_DATE"`
	PhysicalDate string `json:"PHYSICAL_RELEASE_DATE"`
}

func (a *album) String() string {
	if a == nil {
		return ""
	}
	var d string
	if chunks := strings.Split(a.Date, "-"); len(chunks) > 1 {
		d = chunks[0] + " ǁ "
	} else if chunks = strings.Split(a.PhysicalDate, "-"); len(chunks) > 1 {
		d = chunks[0] + " ǁ "
	}

	return d + a.Title
}

type searchResponse struct {
	Track struct {
		Data []struct {
			SongID   string   `json:"SNG_ID"`
			Title    string   `json:"SNG_TITLE"`
			Duration duration `json:"DURATION"`
			Version  string   `json:"VERSION"`
			Artist   string   `json:"ART_NAME"`
			Artists  []struct {
				Name string `json:"ART_NAME"`
			} `json:"ARTISTS"`
			AlbumID    string `json:"ALB_ID"`
			AlbumTitle string `json:"ALB_TITLE"`
		} `json:"data"`
	} `json:"TRACK"`
	Album struct {
		Data []album `json:"data"`
	} `json:"ALBUM"`
}

func (sr searchResponse) tracks() []playlists.Track {
	albums := make(map[string]*album, len(sr.Album.Data))
	for _, a := range sr.Album.Data {
		albums[a.ID] = &a
	}

	out := make([]playlists.Track, 0, len(sr.Track.Data))
	for _, t := range sr.Track.Data {
		if !validID(t.SongID) {
			continue
		}

		artists := []string{t.Artist}
		for _, a := range t.Artists {
			artists = append(artists, a.Name)
		}
		artist := strings.Join(slices.Compact(artists), ", ")

		title := t.Title
		if t.Version != "" {
			title += " " + t.Version
		}

		alb := albums[t.AlbumID].String()
		if alb == "" {
			alb = t.AlbumTitle
		}

		out = append(out, playlists.Track{
			ID:   t.SongID,
			Name: fmt.Sprintf("%s - %s [%s] %s", artist, title, t.Duration, alb),
		})
	}
	return out
}

func (c *Client) SearchTracks(ctx context.Context, query string) (tracks []playlists.Track, err error) {
	tr := c.log.Trace("deezer.SearchTracks").Begins(logs.Var("query", query))
	defer func() { tr.Ends(err, logs.Var("tracks", tracks)) }()

	token, cookies, err := c.tokenizer(ctx)
	if err != nil {
		return nil, err
	}

	in := map[string]any{
		"nb":    100,
		"query": query,
	}

	var out searchResponse
	if _, err := c.send(ctx, tr, token, "deezer.pageSearch", cookies, in, &out); err != nil {
		return nil, err
	}

	return out.tracks(), nil
}

func (c *Client) CreatePlaylist(ctx context.Context, title string) (id string, err error) {
	tr := c.log.Trace("deezer.CreatePlaylist").Begins(logs.Var("title", title))
	defer func() { tr.Ends(err, logs.Var("id", id)) }()

	token, cookies, err := c.tokenizer(ctx)
	if err != nil {
		return "", err
	}

	in := map[string]any{"title": title}

	var out json.Number
	if _, err := c.send(ctx, tr, token, "playlist.create", cookies, in, &out); err != nil {
		return "", err
	}

	if !validID(out.String()) {
		return "", errors.New("failed to create playlist")
	}

	return out.String(), nil
}

func (c *Client) PopulatePlaylist(ctx context.Context, playlist string, tracks []string) (err error) {
	tr := c.log.Trace("deezer.PopulatePlaylist").Begins(logs.Var("playlist", playlist), logs.Var("tracks", tracks))
	defer func() { tr.Ends(err) }()

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
	if _, err := c.send(ctx, tr, token, "playlist.addSongs", cookies, in, &out); err != nil {
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

func (c *Client) send(ctx context.Context, trace *logs.Trace, token, method string, cookies cookieJar, in, out any) (cookieJar, error) {
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

	trace.Dump(method, raw)

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
