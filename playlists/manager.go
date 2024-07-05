package playlists

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

var ErrTrackNotFound = errors.New("track not found")

type Target interface {
	Name() string
	Setup(ctx context.Context) error
	SearchTrack(ctx context.Context, query string) (trackID string, err error)
	CreatePlaylist(ctx context.Context, name string) (playlistID string, err error)
	PopulatePlaylist(ctx context.Context, playlistID string, tracks []string) error
}

type Manager struct {
	target         Target
	maxConcurrency int
	warns          chan string
}

func NewManager(target Target, maxConcurrency int) *Manager {
	return &Manager{
		target:         target,
		maxConcurrency: maxConcurrency,
		warns:          make(chan string),
	}
}

func (m *Manager) Warnings() <-chan string {
	return m.warns
}

func (m *Manager) notify(msg string) {
	select {
	case m.warns <- msg:
	default:
	}
}

type Data struct {
	name   string
	tracks []string
}

func (d *Data) Length() int {
	return len(d.tracks)
}

func (m *Manager) Gather(ctx context.Context, name string, songs []string) (*Data, error) {
	if err := m.target.Setup(ctx); err != nil {
		return nil, fmt.Errorf("%s: setup: %w", m.target.Name(), err)
	}

	defer close(m.warns)

	tracks := newCollection(len(songs))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(m.maxConcurrency)

	for i, song := range songs {
		g.Go(func() error {
			trackID, err := m.target.SearchTrack(ctx, song)
			if errors.Is(err, ErrTrackNotFound) {
				m.notify(fmt.Sprintf("track %q not found", song))
				return nil
			} else if err != nil {
				return fmt.Errorf("%s: searching track %q: %w", m.target.Name(), song, err)
			}

			if err := tracks.add(i, trackID, song); err != nil {
				m.notify(err.Error())
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if tracks.empty() {
		return nil, errors.New("no tracks found")
	}

	return &Data{
		name:   name,
		tracks: tracks.values(),
	}, nil
}

func (m *Manager) Push(ctx context.Context, data *Data) error {
	playlistID, err := m.target.CreatePlaylist(ctx, data.name)
	if err != nil {
		return fmt.Errorf("%s: create playlist: %w", m.target.Name(), err)
	}

	if err := m.target.PopulatePlaylist(ctx, playlistID, data.tracks); err != nil {
		return fmt.Errorf("%s: populate playlist: %w", m.target.Name(), err)
	}

	return nil
}

type collection struct {
	list []string
	set  map[string]string
	mu   sync.RWMutex
}

func newCollection(size int) *collection {
	return &collection{
		list: make([]string, size),
		set:  make(map[string]string, size),
	}
}

func (c *collection) add(i int, id string, name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v, ok := c.set[id]; ok {
		return fmt.Errorf("track %s duplicated, first time %q, now with %q", id, v, name)
	}

	c.list[i] = id
	c.set[id] = name

	return nil
}

func (c *collection) empty() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.list) == 0
}

func (c *collection) values() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]string, 0, len(c.list))
	for _, v := range c.list {
		if v != "" {
			out = append(out, v)
		}
	}

	return out
}
