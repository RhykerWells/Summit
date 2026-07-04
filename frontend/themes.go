package frontend

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

var Themes []Theme

type Theme struct {
	ID string
}

// LoadThemes reads embedded static FS and finds all theme CSS files.
func LoadThemes(static fs.FS) error {
	err := fs.WalkDir(static, "static/css/themes", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		name := filepath.Base(path)

		if filepath.Ext(name) != ".css" {
			return nil
		}

		if name == "core.css" {
			return nil
		}

		Themes = append(Themes, Theme{
			ID: strings.TrimSuffix(name, ".css"),
		})

		return nil
	})

	if err != nil {
		return err
	}

	sort.Slice(Themes, func(i, j int) bool {
		return Themes[i].ID < Themes[j].ID
	})

	return nil
}
