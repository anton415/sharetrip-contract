package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	root, err := readYAML("api/contract.yaml")
	if err != nil {
		panic(err)
	}
	paths := root["paths"].(map[string]any)
	for path, item := range paths {
		ref := item.(map[string]any)["$ref"].(string)
		pathItem, err := readYAML(filepath.Join("api", ref))
		if err != nil {
			panic(err)
		}
		paths[path] = replaceRootReferences(pathItem)
	}
	encoded, err := yaml.Marshal(root)
	if err != nil {
		panic(fmt.Errorf("encode OpenAPI bundle: %w", err))
	}
	_, _ = os.Stdout.Write(encoded)
}

func readYAML(path string) (map[string]any, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, fmt.Errorf("decode %s: %w", path, err)
	}
	return document, nil
}

func replaceRootReferences(value any) any {
	switch item := value.(type) {
	case map[string]any:
		for key, child := range item {
			item[key] = replaceRootReferences(child)
		}
	case []any:
		for index, child := range item {
			item[index] = replaceRootReferences(child)
		}
	case string:
		return strings.ReplaceAll(item, "../contract.yaml#", "#")
	}
	return value
}
