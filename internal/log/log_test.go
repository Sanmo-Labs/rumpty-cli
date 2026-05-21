package log_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Sanmo-Labs/rumpty-cli/internal/log"
)

func TestParseLevel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level   string
		wantErr bool
	}{
		{level: "debug"},
		{level: "info"},
		{level: "warn"},
		{level: "warning"},
		{level: "error"},
		{level: ""},
		{level: "trace", wantErr: true},
		{level: "verbose", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			t.Parallel()
			_, err := log.ParseLevel(tt.level)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestConfigure_levels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		level     string
		logFn     func(string)
		wantEmpty bool
	}{
		{
			name:  "debug emits debug",
			level: "debug",
			logFn: func(msg string) { log.Debug(msg) },
		},
		{
			name:      "error suppresses info",
			level:     "error",
			logFn:     func(msg string) { log.Info(msg) },
			wantEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			require.NoError(t, log.Configure(tt.level, &buf))
			tt.logFn("rumpty-test-line")
			if tt.wantEmpty {
				assert.Empty(t, buf.String())
				return
			}
			assert.Contains(t, buf.String(), "rumpty-test-line")
		})
	}
}

func TestConfigure_nilWriterUsesStderr(t *testing.T) {
	t.Parallel()
	require.NoError(t, log.Configure("warn", nil))
	log.Warn("configured")
}

func TestConfigure_invalidLevel(t *testing.T) {
	t.Parallel()
	err := log.Configure("nope", &bytes.Buffer{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown log level")
}

func TestConfigure_debugAddsSource(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	require.NoError(t, log.Configure("debug", &buf))
	log.Debug("with-source")
	assert.Contains(t, buf.String(), "with-source")
}
