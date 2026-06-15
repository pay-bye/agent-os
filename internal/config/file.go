package config

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

type configFile struct {
	Version      int    `yaml:"version"`
	DatabaseURL  string `yaml:"database_url"`
	Listen       string `yaml:"listen"`
	Declaration  string `yaml:"declaration"`
	GraceSeconds int    `yaml:"shutdown_grace_seconds"`
}

func readConfigFile(path string) (configFile, error) {
	if path == "" {
		return configFile{}, nil
	}
	content, err := readFile(path)
	if err != nil {
		return configFile{}, err
	}
	if err := rejectDuplicateKeys(content); err != nil {
		return configFile{}, err
	}
	var file configFile
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&file); err != nil {
		return configFile{}, fmt.Errorf("unknown_field: %w", err)
	}
	if file.Version != 1 {
		return configFile{}, errors.New("invalid_config_version")
	}
	return file, nil
}

func rejectDuplicateKeys(content []byte) error {
	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		return err
	}
	if len(root.Content) == 0 {
		return nil
	}
	return rejectDuplicates(root.Content[0])
}

func rejectDuplicates(node *yaml.Node) error {
	if node.Kind == yaml.AliasNode || node.Anchor != "" {
		return errors.New("invalid_yaml_feature")
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	seen := map[string]bool{}
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index].Value
		if seen[key] {
			return fmt.Errorf("duplicate_key: %s", key)
		}
		seen[key] = true
		if err := rejectDuplicates(node.Content[index+1]); err != nil {
			return err
		}
	}
	return nil
}
