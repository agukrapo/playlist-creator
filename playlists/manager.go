package playlists

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/agukrapo/playlist-creator/internal/results"
	"golang.org/x/sync/errgroup"
)

var ErrTrackNotFound = errors.New("track not found")

type Track struct {
	ID, Name string
}

type Target interface {
	Name() string
	Setup(ctx context.Context) error
	SearchTracks(ctx context.Context, query string) (matches []Track, err error)
	CreatePlaylist(ctx context.Context, name string) (playlistID string, err error)
	PopulatePlaylist(ctx context.Context, playlistID string, tracks []string) error
}

type Manager struct {
	target         Target
	maxConcurrency int
}

func NewManager(target Target, maxConcurrency int) *Manager {
	return &Manager{
		target:         target,
		maxConcurrency: maxConcurrency,
	}
}

type Callback func(i int, item results.Item, matches []Track)

func (m *Manager) Gather(ctx context.Context, songs []results.Item, fn Callback) error {
	if err := m.target.Setup(ctx); err != nil {
		return fmt.Errorf("%s: setup: %w", m.target.Name(), err)
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(m.maxConcurrency)

	var count atomic.Uint64

	for i, song := range songs {
		g.Go(func() error {
			if song.Active() {
				count.Add(1)
				fn(i, song, nil)
				return nil
			}

			query := song.Query()
			matches, err := m.target.SearchTracks(ctx, query)
			if err != nil {
				return fmt.Errorf("%s: searching %q: %w", m.target.Name(), query, err)
			}

			count.Add(uint64(len(matches)))
			fn(i, song, matches)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	if count.Load() == 0 {
		return errors.New("no tracks found")
	}

	return nil
}

func (m *Manager) Push(ctx context.Context, name string, songs []string) error {
	playlistID, err := m.target.CreatePlaylist(ctx, name)
	if err != nil {
		return fmt.Errorf("%s: create playlist: %w", m.target.Name(), err)
	}

	if err := m.target.PopulatePlaylist(ctx, playlistID, songs); err != nil {
		return fmt.Errorf("%s: populate playlist: %w", m.target.Name(), err)
	}

	return nil
}
