package declaration

import (
	"os"
	"path/filepath"
)

func Read(path string) (Document, error) {
	if path == "" {
		path = DefaultPath
	}
	content, err := readFile(path)
	if err != nil {
		return Document{}, err
	}
	return Parse(content)
}

func readFile(path string) ([]byte, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return root.ReadFile(filepath.Base(absolute))
}
