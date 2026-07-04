package frontend

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type Theme struct {
	ID string
}

// LoadThemes reads embedded static FS and finds all theme CSS files.
func LoadThemes(static fs.FS) ([]Theme, error) {
	var themes []Theme

	err := fs.WalkDir(static, "static/css/themes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := filepath.Base(path)

		if name == "core.css" {
			return nil
		}

		if filepath.Ext(name) != ".css" {
			return nil
		}

		id := strings.TrimSuffix(filepath.Base(path), ".css")

		themes = append(themes, Theme{ID: id})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(themes, func(i, j int) bool {
		return themes[i].ID < themes[j].ID
	})

	return themes, nil
}
