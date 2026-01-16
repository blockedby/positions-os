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

func TestConvertToPgx5URL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "postgres scheme",
			input:    "postgres://user:pass@localhost:5432/db?sslmode=disable",
			expected: "pgx5://user:pass@localhost:5432/db?sslmode=disable",
		},
		{
			name:     "postgresql scheme",
			input:    "postgresql://user:pass@localhost:5432/db",
			expected: "pgx5://user:pass@localhost:5432/db",
		},
		{
			name:     "already pgx5",
			input:    "pgx5://user:pass@localhost:5432/db",
			expected: "pgx5://user:pass@localhost:5432/db",
		},
		{
			name:     "other scheme unchanged",
			input:    "mysql://user:pass@localhost:3306/db",
			expected: "mysql://user:pass@localhost:3306/db",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToPgx5URL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
