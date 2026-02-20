package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratePassword_CorrectLength(t *testing.T) {
	tests := []struct {
		name   string
		length int
		want   int
	}{
		{"standard length", 16, 16},
		{"short", 4, 4},
		{"long", 64, 64},
		{"single char", 1, 1},
		{"zero defaults to 16", 0, defaultPasswordLength},
		{"negative defaults to 16", -5, defaultPasswordLength},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pw, err := GeneratePassword(tt.length)
			require.NoError(t, err)
			assert.Len(t, pw, tt.want)
		})
	}
}

func TestGeneratePassword_ValidChars(t *testing.T) {
	pw, err := GeneratePassword(200)
	require.NoError(t, err)

	for _, ch := range pw {
		assert.Contains(t, passwordChars, string(ch), "unexpected character: %c", ch)
	}
}

func TestGeneratePassword_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		pw, err := GeneratePassword(32)
		require.NoError(t, err)
		assert.False(t, seen[pw], "duplicate password generated")
		seen[pw] = true
	}
}
