package results

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSet_Add(t *testing.T) {
	t.Run("same index, different ids", func(t *testing.T) {
		s := New(1)

		require.True(t, s.Add(0, "_A"))
		require.True(t, s.Add(0, "_B"))
		require.True(t, s.Add(0, "_C"))

		assert.Equal(t, []string{"_C"}, s.Slice())
	})
	t.Run("different index, same ids", func(t *testing.T) {
		s := New(3)

		require.True(t, s.Add(0, "_A"))
		require.False(t, s.Add(1, "_A"))
		require.False(t, s.Add(2, "_A"))

		assert.Equal(t, []string{"_A"}, s.Slice())
	})
	t.Run("different index, different ids", func(t *testing.T) {
		s := New(3)

		require.True(t, s.Add(0, "_A"))
		require.True(t, s.Add(1, "_B"))
		require.True(t, s.Add(2, "_C"))

		assert.Equal(t, []string{"_A", "_B", "_C"}, s.Slice())
	})
	t.Run("missing middle index", func(t *testing.T) {
		s := New(4)

		require.True(t, s.Add(0, "_A"))
		require.True(t, s.Add(2, "_C"))

		assert.Equal(t, []string{"_A", "_C"}, s.Slice())
	})
	t.Run("should purge old values", func(t *testing.T) {
		s := New(4)

		require.True(t, s.Add(0, "_A"))
		require.True(t, s.Add(0, "_B"))
		require.True(t, s.Add(0, "_A"))

		assert.Equal(t, []string{"_A"}, s.Slice())
	})
}
