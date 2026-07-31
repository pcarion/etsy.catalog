package catalog

import (
	"fmt"
	"os"
	"path/filepath"

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
	b, err := yaml.Marshal(d)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "listing.yaml"), b, 0644)
}
