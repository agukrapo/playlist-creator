package playlists

import (
	"context"
	"errors"
	"fmt"

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
	target Target
}

func NewManager(target Target) *Manager {
	return &Manager{
		target: target,
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

	tracks := make([]string, len(songs))

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(100)

	for i, song := range songs {
		g.Go(func() error {
			trackID, err := m.target.SearchTrack(ctx, song)
			if errors.Is(err, ErrTrackNotFound) {
				fmt.Printf("track %q not found\n", song)
			} else if err != nil {
				return fmt.Errorf("%s: searching track %q: %w", m.target.Name(), song, err)
			}

			tracks[i] = trackID
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if len(tracks) == 0 {
		return nil, errors.New("no tracks found")
	}

	return &Data{
		name:   name,
		tracks: tracks,
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
