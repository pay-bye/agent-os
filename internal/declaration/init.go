package declaration

import (
	"errors"
	"os"
	"path/filepath"
)

var ErrMissingExplicitVocabularyInput = errors.New("missing_explicit_vocabulary_input")

type InitInput struct {
	Path string
	Yes  bool
}

func Init(input InitInput) error {
	if input.Yes {
		return ErrMissingExplicitVocabularyInput
	}
	path := input.Path
	if path == "" {
		path = DefaultPath
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o750); err != nil {
		return err
	}
	root, err := os.OpenRoot(filepath.Dir(absolute))
	if err != nil {
		return err
	}
	defer root.Close()
	file, err := root.OpenFile(filepath.Base(absolute), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return errors.New("declaration_exists")
		}
		return err
	}
	defer file.Close()

	_, err = file.WriteString(neutralDocument())
	return err
}

func neutralDocument() string {
	return `version: 1
schemas: {}
items: {}
needs: {}
nodes: {}
routes: {}
`
}
