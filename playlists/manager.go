package playlists

import (
	"context"
	"errors"
	"fmt"
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

func (m *Manager) Start(ctx context.Context, name string, songs []string) error {
	if err := m.target.Setup(ctx); err != nil {
		return fmt.Errorf("%s: setup: %w", m.target.Name(), err)
	}

	tracks := make([]string, 0, len(songs))
	for _, song := range songs {
		trackID, err := m.target.SearchTrack(ctx, song)
		if errors.Is(err, ErrTrackNotFound) {
			fmt.Printf("%s: %v", song, err)
		} else if err != nil {
			return fmt.Errorf("%s: search track: %w", m.target.Name(), err)
		}

		tracks = append(tracks, trackID)
	}

	if len(tracks) == 0 {
		return errors.New("no tracks found")
	}

	fmt.Printf("Creating playlist %q with %d tracks\nPress the Enter Key to continue\n", name, len(tracks))

	if _, err := fmt.Scanln(); err != nil {
		return err
	}

	playlistID, err := m.target.CreatePlaylist(ctx, name)
	if err != nil {
		return fmt.Errorf("%s: create playlist: %w", m.target.Name(), err)
	}

	if err := m.target.PopulatePlaylist(ctx, playlistID, tracks); err != nil {
		return fmt.Errorf("%s: populate playlist: %w", m.target.Name(), err)
	}

	return nil
}
