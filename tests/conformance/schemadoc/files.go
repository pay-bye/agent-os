package schemadoc

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func FindRoot(t testing.TB, segments ...string) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		root := filepath.Join(append([]string{dir}, segments...)...)
		if Exists(root) {
			return root
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("%s root not found", filepath.Join(segments...))
		}
		dir = parent
	}
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !errors.Is(err, os.ErrNotExist)
}

func FilesUnder(root string) ([]string, error) {
	items := []string{}
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			items = append(items, path)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return items, nil
}
