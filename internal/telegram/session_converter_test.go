package telegram

import (
	"encoding/json"
	"testing"

	"github.com/gotd/td/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertToGotgprotoSession_Success(t *testing.T) {
	// Arrange
	input := &session.Data{
		DC:      2,
		Addr:    "149.154.167.40:443",
		AuthKey: []byte("test-auth-key-32-bytes-long-abc"),
	}

	// Act
	result, err := ConvertToGotgprotoSession(input)

	// Assert
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.NotEmpty(t, result.Data, "Data should be populated")

	// Verify JSON structure
	var parsed map[string]interface{}
	err = json.Unmarshal(result.Data, &parsed)
	require.NoError(t, err, "Data should be valid JSON")
	assert.Equal(t, float64(2), parsed["DC"])
}

func TestConvertToGotgprotoSession_NilInput(t *testing.T) {
	// Act
	// This captures the likely panic/error if input is nil
	result, err := ConvertToGotgprotoSession(nil)

	// Assert
	if err == nil {
		t.Error("Expected error for nil input, got nil")
	}
	assert.Nil(t, result)
}
