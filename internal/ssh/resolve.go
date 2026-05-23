package ssh

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/Sanmo-Labs/rumpty-cli/internal/app"
	"github.com/Sanmo-Labs/rumpty-cli/internal/vm"
)

func resolveVMRef(ctx context.Context, rt *app.Runtime, ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("vm name or slug is required")
	}
	if needsListLookup(ref) {
		return resolveVMSlug(ctx, rt, ref)
	}
	return ref, nil
}

func resolveVMSlug(ctx context.Context, rt *app.Runtime, ref string) (string, error) {
	target, err := vm.Find(ctx, rt, ref)
	if err != nil {
		return "", err
	}
	return target.Slug, nil
}

func needsListLookup(ref string) bool {
	if strings.Contains(ref, " ") {
		return true
	}
	return looksLikeUID(ref)
}

func looksLikeUID(ref string) bool {
	if len(ref) < 32 {
		return false
	}
	dashes := 0
	for _, r := range ref {
		switch {
		case unicode.IsDigit(r), r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		case r == '-':
			dashes++
		default:
			return false
		}
	}
	return dashes >= 4
}
