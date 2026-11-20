package golog

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zapcore"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
		{Level(99), "INVALID(99)"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.String())
	}
}

func TestLevelZapLevel(t *testing.T) {
	tests := []struct {
		level    Level
		expected zapcore.Level
	}{
		{DebugLevel, zapcore.DebugLevel},
		{InfoLevel, zapcore.InfoLevel},
		{WarnLevel, zapcore.WarnLevel},
		{ErrorLevel, zapcore.ErrorLevel},
		{FatalLevel, zapcore.FatalLevel},
		{Level(99), zapcore.InvalidLevel},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expected, tt.level.ZapLevel())
	}
}

func TestLevelEnabled(t *testing.T) {
	assert.True(t, InfoLevel.Enabled(DebugLevel))
	assert.True(t, InfoLevel.Enabled(InfoLevel))
	assert.False(t, InfoLevel.Enabled(WarnLevel))
	assert.False(t, Level(99).Enabled(DebugLevel))
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
		wantErr  bool
	}{
		{"DEBUG", DebugLevel, false},
		{"debug", DebugLevel, false},
		{" Info ", InfoLevel, false},
		{"WARN", WarnLevel, false},
		{"error", ErrorLevel, false},
		{"FATAL", FatalLevel, false},
		{"UNKNOWN", InfoLevel, true},
		{"", InfoLevel, true},
	}

	for _, tt := range tests {
		lvl, err := ParseLevel(tt.input)
		if tt.wantErr {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, lvl)
		}
	}
}
