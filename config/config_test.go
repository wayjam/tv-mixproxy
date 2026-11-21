package config

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFlexInt(t *testing.T) {
	type Sample struct {
		Val FlexInt `json:"val"`
	}

	tests := []struct {
		input    string
		expected int
	}{
		{`{"val": 1}`, 1},
		{`{"val": "2"}`, 2},
		{`{"val": true}`, 1},
		{`{"val": false}`, 0},
		{`{"val": "true"}`, 1},
		{`{"val": "false"}`, 0},
		{`{"val": null}`, 0},
		{`{"val": 0}`, 0},
		{`{"val": 3.0}`, 3},
	}

	for _, tt := range tests {
		var s Sample
		err := json.Unmarshal([]byte(tt.input), &s)
		assert.NoError(t, err, "failed on input: %s", tt.input)
		assert.Equal(t, tt.expected, int(s.Val), "mismatch for input: %s", tt.input)
	}
}

func TestStripJSONComments(t *testing.T) {
	input := `
	// Top line comment
	{
		"url": "http://example.com/api?foo=//bar", // inline comment
		/* block comment */
		"key": "value/*not comment*/",
		"active": true // another inline
	}
	`
	cleaned := StripJSONComments([]byte(input))

	var res map[string]interface{}
	err := json.Unmarshal(cleaned, &res)
	assert.NoError(t, err, "should parse as valid JSON after stripping comments")
	assert.Equal(t, "http://example.com/api?foo=//bar", res["url"])
	assert.Equal(t, "value/*not comment*/", res["key"])
	assert.Equal(t, true, res["active"])
}
