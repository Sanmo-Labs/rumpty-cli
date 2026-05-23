package ssh_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	rumptyssh "github.com/Sanmo-Labs/rumpty-cli/internal/ssh"
)

func TestNeedsListLookup(t *testing.T) {
	t.Parallel()

	assert.False(t, rumptyssh.NeedsListLookupForTest("test-vm7"))
	assert.False(t, rumptyssh.NeedsListLookupForTest("warm-jollof"))
	assert.True(t, rumptyssh.NeedsListLookupForTest("Test VM Seven"))
	assert.True(t, rumptyssh.NeedsListLookupForTest("019e2b95-1234-5678-9abc-def012345678"))
}
