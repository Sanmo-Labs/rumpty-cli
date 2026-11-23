package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
)

// Find resolves a database by UID, slug or name within the configured workspace.
func Find(ctx context.Context, rt *app.Runtime, ref string) (api.Database, error) {
	ref = strings.TrimSpace(ref)
	workspace := strings.TrimSpace(rt.Config.Workspace)
	databases, err := rt.API().ListDatabases(ctx, workspace)
	if err != nil {
		return api.Database{}, err
	}

	var matches []api.Database
	for i := range databases {
		d := &databases[i]
		if d.UID == ref || d.Slug == ref || d.Name == ref {
			matches = append(matches, *d)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], nil
	case 0:
		return api.Database{}, fmt.Errorf("database %q not found in workspace %s", ref, workspace)
	default:
		names := make([]string, 0, len(matches))
		for i := range matches {
			names = append(names, matches[i].Slug)
		}
		return api.Database{}, fmt.Errorf("multiple databases match %q: %s", ref, strings.Join(names, ", "))
	}
}
