package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pcarion/etsy.catalog/internal/catalog"
	"github.com/spf13/cobra"
)

var writable = map[string]bool{
	"quantity": true, "title": true, "description": true, "price": true, "who_made": true, "when_made": true, "taxonomy_id": true,
	"shipping_profile_id": true, "readiness_state_id": true, "materials": true, "shop_section_id": true, "production_partner_ids": true,
	"tags": true, "type": true, "is_supply": true, "is_customizable": true, "should_auto_renew": true, "is_taxable": true, "state": true,
}

func newPushCmd() *cobra.Command {
	var dryRun bool
	c := &cobra.Command{Use: "push <catalog-id>", Short: "Create or update an Etsy listing from local YAML", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		d, dir, err := catalog.Load(opts.root, args[0])
		if err != nil {
			return err
		}
		form := listingForm(d.Listing)
		if err := validateCreate(d.EtsyID, form); err != nil {
			return err
		}
		if dryRun {
			fmt.Fprintf(opts.out, "Would %s listing %d with %d fields, %d images, %d files\n", map[bool]string{true: "update", false: "create"}[d.EtsyID > 0], d.EtsyID, len(form), len(d.Images), len(d.Files))
			return nil
		}
		if err := requireShop(); err != nil {
			return err
		}
		cl, err := client()
		if err != nil {
			return err
		}
		id := d.EtsyID
		desiredState := form.Get("state")
		if id == 0 {
			form.Del("state")
		}
		if id == 0 {
			var result map[string]any
			if err := cl.DoForm(cmd.Context(), http.MethodPost, fmt.Sprintf("shops/%s/listings", opts.shopID), form, &result); err != nil {
				return err
			}
			id = int64num(result["listing_id"])
			if id == 0 {
				return fmt.Errorf("Etsy create response did not contain listing_id")
			}
			d.EtsyID = id
		} else {
			if err := cl.DoForm(cmd.Context(), http.MethodPatch, fmt.Sprintf("shops/%s/listings/%d", opts.shopID, id), form, nil); err != nil {
				return err
			}
		}
		if len(d.Inventory) > 0 {
			if err := cl.Do(cmd.Context(), http.MethodPut, fmt.Sprintf("listings/%d/inventory", id), nil, d.Inventory, nil); err != nil {
				return fmt.Errorf("update inventory: %w", err)
			}
		}
		for i := range d.Images {
			a := &d.Images[i]
			if a.Path == "" || a.ID != 0 {
				continue
			}
			var result map[string]any
			fields := map[string]string{}
			if a.Rank > 0 {
				fields["rank"] = strconv.Itoa(a.Rank)
			}
			if a.Alt != "" {
				fields["alt_text"] = a.Alt
			}
			if err := cl.Upload(cmd.Context(), fmt.Sprintf("shops/%s/listings/%d/images", opts.shopID, id), "image", filepath.Join(dir, a.Path), fields, &result); err != nil {
				return fmt.Errorf("upload image %s: %w", a.Path, err)
			}
			a.ID = int64num(result["listing_image_id"])
		}
		for i := range d.Files {
			a := &d.Files[i]
			if a.Path == "" || a.ID != 0 {
				continue
			}
			var result map[string]any
			if err := cl.Upload(cmd.Context(), fmt.Sprintf("shops/%s/listings/%d/files", opts.shopID, id), "file", filepath.Join(dir, a.Path), nil, &result); err != nil {
				return fmt.Errorf("upload file %s: %w", a.Path, err)
			}
			a.ID = int64num(result["listing_file_id"])
		}
		if desiredState != "" && form.Get("state") == "" {
			if err := cl.DoForm(cmd.Context(), http.MethodPatch, fmt.Sprintf("shops/%s/listings/%d", opts.shopID, id), url.Values{"state": {desiredState}}, nil); err != nil {
				return fmt.Errorf("set listing state: %w", err)
			}
		}
		if err := catalog.Save(opts.root, args[0], d); err != nil {
			return err
		}
		fmt.Fprintf(opts.out, "Pushed catalog %s to Etsy listing %d\n", args[0], id)
		return nil
	}}
	c.Flags().BoolVar(&dryRun, "dry-run", false, "validate and describe changes without calling Etsy")
	return c
}

func listingForm(m map[string]any) url.Values {
	f := url.Values{}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !writable[k] || m[k] == nil {
			continue
		}
		switch v := m[k].(type) {
		case []any:
			parts := make([]string, len(v))
			for i := range v {
				parts[i] = fmt.Sprint(v[i])
			}
			f.Set(k, strings.Join(parts, ","))
		case []string:
			f.Set(k, strings.Join(v, ","))
		case map[string]any:
			if k == "price" {
				f.Set(k, moneyValue(v))
			} else {
				b, _ := json.Marshal(v)
				f.Set(k, string(b))
			}
		default:
			f.Set(k, fmt.Sprint(v))
		}
	}
	return f
}

func moneyValue(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return fmt.Sprint(v)
	}
	amount, aok := number(m["amount"])
	divisor, dok := number(m["divisor"])
	if !aok || !dok || divisor == 0 {
		return fmt.Sprint(v)
	}
	return strconv.FormatFloat(amount/divisor, 'f', -1, 64)
}

func number(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case string:
		f, e := strconv.ParseFloat(n, 64)
		return f, e == nil
	}
	return 0, false
}
func validateCreate(id int64, f url.Values) error {
	if id > 0 {
		return nil
	}
	required := []string{"quantity", "title", "description", "price", "who_made", "when_made", "taxonomy_id"}
	missing := []string{}
	for _, k := range required {
		if f.Get(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("new listing is missing required fields: %s", strings.Join(missing, ", "))
	}
	return nil
}
