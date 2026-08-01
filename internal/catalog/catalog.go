package catalog

import (
	"fmt"
	"html"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Document struct {
	CatalogID string         `yaml:"catalog_id"`
	EtsyID    int64          `yaml:"etsy_listing_id,omitempty"`
	Listing   map[string]any `yaml:"listing"`
	Inventory map[string]any `yaml:"inventory,omitempty"`
	Images    []Asset        `yaml:"images,omitempty"`
	Files     []Asset        `yaml:"files,omitempty"`
}

type Asset struct {
	ID       int64          `yaml:"etsy_id,omitempty"`
	Path     string         `yaml:"path,omitempty"`
	Rank     int            `yaml:"rank,omitempty"`
	Alt      string         `yaml:"alt_text,omitempty"`
	Metadata map[string]any `yaml:"metadata,omitempty"`
}

func Dir(root, id string) string { return filepath.Join(root, id) }

func Load(root, id string) (*Document, string, error) {
	dir := Dir(root, id)
	name := filepath.Join(dir, "listing.yaml")
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, dir, fmt.Errorf("read %s: %w", name, err)
	}
	var d Document
	if err := yaml.Unmarshal(b, &d); err != nil {
		return nil, dir, fmt.Errorf("parse %s: %w", name, err)
	}
	if d.CatalogID == "" {
		d.CatalogID = id
	}
	if d.Listing == nil {
		return nil, dir, fmt.Errorf("%s: listing is required", name)
	}
	return &d, dir, nil
}

func Save(root, id string, d *Document) error {
	dir := Dir(root, id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	var node yaml.Node
	if err := node.Encode(d); err != nil {
		return err
	}
	formatYAML(&node, "")
	b, err := yaml.Marshal(&node)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "listing.yaml"), b, 0644)
}

// formatYAML makes the generated catalog pleasant to edit while preserving its
// normal YAML types when it is loaded again.
func formatYAML(node *yaml.Node, key string) {
	switch node.Kind {
	case yaml.DocumentNode:
		for _, child := range node.Content {
			formatYAML(child, "")
		}
	case yaml.MappingNode:
		for i := 0; i+1 < len(node.Content); i += 2 {
			formatYAML(node.Content[i+1], node.Content[i].Value)
		}
	case yaml.SequenceNode:
		for _, child := range node.Content {
			formatYAML(child, key)
		}
	case yaml.ScalarNode:
		if node.Tag == "!!str" {
			node.Value = html.UnescapeString(node.Value)
		}
		if node.Tag == "!!str" && strings.Contains(node.Value, "\n") {
			lines := strings.Split(node.Value, "\n")
			for i := range lines {
				lines[i] = strings.TrimRight(lines[i], " \t")
			}
			node.Value = strings.Join(lines, "\n")
			node.Style = yaml.LiteralStyle
		}
		if node.Tag != "!!float" {
			return
		}
		value, err := strconv.ParseFloat(node.Value, 64)
		if err != nil || math.Trunc(value) != value {
			return
		}
		switch {
		case strings.HasSuffix(key, "_id"):
			node.Tag = "!!int"
			node.Value = strconv.FormatInt(int64(value), 10)
		case strings.HasSuffix(key, "_timestamp"), strings.HasSuffix(key, "_tsz"):
			node.Tag = "!!str"
			node.Value = time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
		}
	}
}
