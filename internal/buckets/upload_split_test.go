package buckets

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitUploadKey_directoryKeepsFilename(t *testing.T) {
	name, prefix := splitUploadKey("reports/", "report.pdf")
	assert.Equal(t, "report.pdf", name)
	assert.Equal(t, "reports", prefix)

	name, prefix = splitUploadKey("assets/reports/", "report.pdf")
	assert.Equal(t, "report.pdf", name)
	assert.Equal(t, "assets/reports", prefix)
}

func TestSplitUploadKey_fileDestination(t *testing.T) {
	name, prefix := splitUploadKey("report.pdf", "x")
	assert.Equal(t, "report.pdf", name)
	assert.Empty(t, prefix)

	name, prefix = splitUploadKey("docs/report.pdf", "x")
	assert.Equal(t, "report.pdf", name)
	assert.Equal(t, "docs", prefix)
}

func TestSplitUploadKey_rootOfBucket(t *testing.T) {
	name, prefix := splitUploadKey("", "report.pdf")
	assert.Equal(t, "report.pdf", name)
	assert.Empty(t, prefix)
}
