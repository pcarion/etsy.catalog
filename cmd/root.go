package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pcarion/etsy.catalog/internal/etsy"
	"github.com/spf13/cobra"
)

type options struct {
	shopID, apiKey, token, root string
	out, err                    io.Writer
}

var opts options

func Execute() error { return newRootCmd().Execute() }

func newRootCmd() *cobra.Command {
	opts = options{apiKey: os.Getenv("ETSY_API_KEY"), token: os.Getenv("ETSY_ACCESS_TOKEN"), shopID: os.Getenv("ETSY_SHOP_ID"), root: "listings", out: os.Stdout, err: os.Stderr}
	c := &cobra.Command{Use: "etsy", Short: "Manage Etsy listings as local YAML", SilenceUsage: true, SilenceErrors: true}
	c.PersistentFlags().StringVar(&opts.shopID, "shop-id", opts.shopID, "Etsy shop ID (or ETSY_SHOP_ID)")
	c.PersistentFlags().StringVar(&opts.apiKey, "api-key", opts.apiKey, "Etsy keystring:shared_secret (or ETSY_API_KEY)")
	c.PersistentFlags().StringVar(&opts.token, "access-token", opts.token, "OAuth access token (or ETSY_ACCESS_TOKEN)")
	c.PersistentFlags().StringVar(&opts.root, "root", opts.root, "catalog directory")
	c.AddCommand(newAuthCmd(), newListCmd(), newGetCmd(), newPushCmd())
	return c
}

func client() (*etsy.Client, error) {
	missing := []string{}
	if opts.apiKey == "" {
		missing = append(missing, "ETSY_API_KEY")
	}
	if opts.token == "" {
		missing = append(missing, "ETSY_ACCESS_TOKEN")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing credentials: %s", strings.Join(missing, ", "))
	}
	return etsy.New(opts.apiKey, opts.token), nil
}

func requireShop() error {
	if opts.shopID == "" {
		return fmt.Errorf("missing shop ID: set ETSY_SHOP_ID or use --shop-id")
	}
	return nil
}
