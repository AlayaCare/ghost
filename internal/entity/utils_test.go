package entity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStringArrayEquivalent(t *testing.T) {
	t.Run("StringArrayEquivalent: no change", func(t *testing.T) {
		res, added, removed := StringArrayEquivalent([]string{"aa", "bb"}, []string{"bb", "aa"})

		assert.Equal(t, true, res)
		assert.Equal(t, 0, len(added))
		assert.Equal(t, 0, len(removed))

	})

	t.Run("StringArrayEquivalent: removed", func(t *testing.T) {
		res, added, removed := StringArrayEquivalent([]string{"aa", "cc", "bb"}, []string{"bb", "aa"})

		assert.Equal(t, false, res)
		assert.Equal(t, 0, len(added))
		assert.Equal(t, 1, len(removed))

	})

	t.Run("StringArrayEquivalent: added", func(t *testing.T) {
		res, added, removed := StringArrayEquivalent([]string{"aa", "bb"}, []string{"bb", "cc", "aa"})

		assert.Equal(t, false, res)
		assert.Equal(t, 1, len(added))
		assert.Equal(t, 0, len(removed))

	})
}
