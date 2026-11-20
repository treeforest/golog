package golog

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigOptions(t *testing.T) {
	cfg := NewConfig(
		WithModule("auth"),
		WithComponent("jwt"),
		WithPath("/tmp/test.log"),
		WithLevel(DebugLevel),
		WithMaxAgeDays(7),
		WithMaxBackups(5),
		WithRotationSizeMB(200),
		WithCompress(true),
		WithJSONFormat(true),
		WithUseUTC(false),
		WithShowLine(false),
		WithLogInFile(true),
		WithLogInConsole(false),
		WithShowColor(true),
		WithBrief(true),
		WithStackTraceLevel(FatalLevel),
		WithSampling(SamplingConfig{
			Enabled:    true,
			Initial:    50,
			Thereafter: 50,
		}),
	)

	assert.Equal(t, "auth", cfg.Module)
	assert.Equal(t, "jwt", cfg.Component)
	assert.Equal(t, "/tmp/test.log", cfg.Path)
	assert.Equal(t, DebugLevel, cfg.Level)
	assert.Equal(t, 7, cfg.MaxAgeDays)
	assert.Equal(t, 5, cfg.MaxBackups)
	assert.Equal(t, int64(200), cfg.RotationSizeMB)
	assert.True(t, cfg.Compress)
	assert.True(t, cfg.JSONFormat)
	assert.False(t, cfg.UseUTC)
	assert.False(t, cfg.ShowLine)
	assert.True(t, cfg.LogInFile)
	assert.False(t, cfg.LogInConsole)
	assert.True(t, cfg.ShowColor)
	assert.True(t, cfg.Brief)
	assert.Equal(t, FatalLevel, cfg.StackTraceLevel)
	assert.True(t, cfg.Sampling.Enabled)
	assert.Equal(t, 50, cfg.Sampling.Initial)
	assert.Equal(t, 50, cfg.Sampling.Thereafter)
}
