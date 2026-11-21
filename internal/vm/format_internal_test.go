package vm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPrettyImage(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Ubuntu 26.04", prettyImage("ubuntu-2604-lts"))
	assert.Equal(t, "Ubuntu 24.04", prettyImage("ubuntu-2404-lts"))
	assert.Equal(t, "Ubuntu 26.04 Desktop", prettyImage("ubuntu-2604-desktop"))
	assert.Equal(t, "debian-12", prettyImage("debian-12"))
	assert.Equal(t, "—", prettyImage("—"))
}

func TestFormatElapsed(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "0s", formatElapsed(200*time.Millisecond))
	assert.Equal(t, "12s", formatElapsed(12400*time.Millisecond))
	assert.Equal(t, "1m04s", formatElapsed(64*time.Second))
}
