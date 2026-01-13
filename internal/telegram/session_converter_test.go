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
	assert.Equal(t, 1, result.Version, "Version should be 1")

	// Verify JSON structure - gotgproto expects wrapped format:
	// {"Version":1,"Data":{"DC":2,"Addr":"...","AuthKey":"...",...}}
	var parsed map[string]interface{}
	err = json.Unmarshal(result.Data, &parsed)
	require.NoError(t, err, "Data should be valid JSON")

	// Check wrapper fields
	assert.Equal(t, float64(1), parsed["Version"], "Should have Version=1")
	assert.Contains(t, parsed, "Data", "Should have Data field")

	// Check nested Data field contains the actual session data
	dataObj, ok := parsed["Data"].(map[string]interface{})
	require.True(t, ok, "Data should be an object")
	assert.Equal(t, float64(2), dataObj["DC"], "DC should be in nested Data")
	assert.Equal(t, "149.154.167.40:443", dataObj["Addr"], "Addr should be in nested Data")
}

func TestConvertToGotgprotoSession_NilInput(t *testing.T) {
	// Act
	result, err := ConvertToGotgprotoSession(nil)

	// Assert
	require.Error(t, err, "Expected error for nil input")
	assert.Nil(t, result)
}
