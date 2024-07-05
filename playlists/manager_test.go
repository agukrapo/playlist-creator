package playlists

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTarget struct {
	Target
}

func (tt testTarget) Name() string {
	return "testTarget"
}

func (tt testTarget) Setup(context.Context) error {
	return nil
}

func (tt testTarget) SearchTrack(_ context.Context, query string) (string, error) {
	if query == "_B" {
		return "", ErrTrackNotFound
	}
	return query, nil
}

func TestManager_Gather(t *testing.T) {
	manager := NewManager(testTarget{}, 100)

	data, err := manager.Gather(context.Background(), "_TEST", []string{"_C", "_B", "_A", "_A"})
	require.NoError(t, err)

	assert.Equal(t, []string{"_C", "_A"}, data.tracks)
}
