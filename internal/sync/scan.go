package sync

import (
	"io/fs"
	"path/filepath"
)

// ScanLocal walks root and returns regular files keyed by slash-separated
// relative path. Symlinks are skipped.
func ScanLocal(root string) (map[string]FileInfo, error) {
	files := make(map[string]FileInfo)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		key := filepath.ToSlash(rel)
		files[key] = FileInfo{Key: key, Size: info.Size(), ModTime: info.ModTime()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
