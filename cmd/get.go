package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pcarion/etsy.catalog/internal/catalog"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{Use: "get <catalog-id>", Short: "Export an Etsy listing to listings/<catalog-id>/listing.yaml", Args: cobra.ExactArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireShop(); err != nil {
			return err
		}
		cl, err := client()
		if err != nil {
			return err
		}
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || id < 1 {
			return fmt.Errorf("catalog-id must be an Etsy numeric listing ID")
		}
		var listing map[string]any
		q := url.Values{"includes": {"Images,Inventory,Videos"}}
		if err := cl.Do(cmd.Context(), http.MethodGet, fmt.Sprintf("listings/%d", id), q, nil, &listing); err != nil {
			return err
		}
		d := &catalog.Document{CatalogID: args[0], EtsyID: id, Listing: listing}
		if inv, ok := listing["inventory"].(map[string]any); ok {
			d.Inventory = writableInventory(inv)
		}
		if images, ok := listing["images"].([]any); ok {
			for _, raw := range images {
				if m, ok := raw.(map[string]any); ok {
					d.Images = append(d.Images, catalog.Asset{ID: int64num(m["listing_image_id"]), Rank: int(int64num(m["rank"])), Alt: textValue(m["alt_text"]), Metadata: m})
				}
			}
		}
		var fileResponse struct {
			Results []map[string]any `json:"results"`
		}
		if err := cl.Do(cmd.Context(), http.MethodGet, fmt.Sprintf("shops/%s/listings/%d/files", opts.shopID, id), nil, nil, &fileResponse); err != nil {
			return fmt.Errorf("get listing files: %w", err)
		}
		for _, m := range fileResponse.Results {
			d.Files = append(d.Files, catalog.Asset{ID: int64num(m["listing_file_id"]), Metadata: m})
		}
		if err := catalog.Save(opts.root, args[0], d); err != nil {
			return err
		}
		fmt.Fprintf(opts.out, "Wrote %s/listing.yaml\n", catalog.Dir(opts.root, args[0]))
		return nil
	}}
}

func int64num(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case int:
		return int64(n)
	}
	return 0
}
func textValue(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func writableInventory(inv map[string]any) map[string]any {
	out := map[string]any{}
	for _, k := range []string{"price_on_property", "quantity_on_property", "sku_on_property", "readiness_state_on_property"} {
		if v, ok := inv[k]; ok {
			out[k] = v
		}
	}
	products, _ := inv["products"].([]any)
	cleanProducts := make([]any, 0, len(products))
	for _, raw := range products {
		p, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cp := map[string]any{"sku": p["sku"], "property_values": p["property_values"]}
		offerings, _ := p["offerings"].([]any)
		cleanOfferings := make([]any, 0, len(offerings))
		for _, ro := range offerings {
			o, ok := ro.(map[string]any)
			if !ok {
				continue
			}
			co := map[string]any{}
			for _, k := range []string{"quantity", "is_enabled", "readiness_state_id"} {
				if v, exists := o[k]; exists {
					co[k] = v
				}
			}
			if v, exists := o["price"]; exists {
				co["price"] = moneyValue(v)
			}
			cleanOfferings = append(cleanOfferings, co)
		}
		cp["offerings"] = cleanOfferings
		cleanProducts = append(cleanProducts, cp)
	}
	out["products"] = cleanProducts
	return out
}
