package migrator

import (
	"context"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithFS(t *testing.T) {
	// Test with valid FS
	fs := fstest.MapFS{
		"0001_test.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"0001_test.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	m, err := NewWithFS(fs)
	require.NoError(t, err)
	assert.NotNil(t, m)
}

func TestNewWithFS_NilFS(t *testing.T) {
	m, err := NewWithFS(nil)
	assert.Error(t, err)
	assert.Nil(t, m)
}

func TestMigrator_Up_InvalidURL(t *testing.T) {
	fs := fstest.MapFS{
		"0001_test.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"0001_test.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	m, err := NewWithFS(fs)
	require.NoError(t, err)

	ctx := context.Background()
	err = m.Up(ctx, "invalid://url")
	assert.Error(t, err)
}

func TestMigrator_Up_EmptyURL(t *testing.T) {
	fs := fstest.MapFS{
		"0001_test.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"0001_test.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	m, err := NewWithFS(fs)
	require.NoError(t, err)

	ctx := context.Background()
	err = m.Up(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}
