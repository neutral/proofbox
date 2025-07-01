package storage_test

import (
	"testing"

	"github.com/neutral/proofbox/pkg/storage/memory"
	"github.com/neutral/proofbox/pkg/storage/pebble"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoubleClose(t *testing.T) {
	t.Run("MemoryStorage", func(t *testing.T) {
		store := memory.NewStorage()
		
		// First close should succeed
		err := store.Close()
		assert.NoError(t, err)
		
		// Second close should not panic or error
		err = store.Close()
		assert.NoError(t, err)
	})

	t.Run("PebbleStorage", func(t *testing.T) {
		tmpDir := t.TempDir()
		store, err := pebble.NewStorage(tmpDir, nil)
		require.NoError(t, err)
		
		// First close should succeed
		err = store.Close()
		assert.NoError(t, err)
		
		// Second close should not panic or error
		err = store.Close()
		assert.NoError(t, err)
	})
}