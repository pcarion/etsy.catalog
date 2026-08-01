package cmd

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func newShopCmd() *cobra.Command {
	var idOnly bool
	c := &cobra.Command{
		Use:   "shop",
		Short: "Get the shop associated with the authenticated Etsy user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cl, err := client()
			if err != nil {
				return err
			}
			userID, err := userIDFromToken(opts.token)
			if err != nil {
				return err
			}
			var shop struct {
				ShopID   int64  `json:"shop_id"`
				ShopName string `json:"shop_name"`
			}
			if err := cl.Do(cmd.Context(), http.MethodGet, fmt.Sprintf("users/%s/shops", userID), nil, nil, &shop); err != nil {
				return err
			}
			if shop.ShopID < 1 {
				return fmt.Errorf("Etsy response did not contain a shop_id")
			}
			if idOnly {
				fmt.Fprintln(opts.out, shop.ShopID)
			} else {
				fmt.Fprintf(opts.out, "ID\tNAME\n%d\t%s\n", shop.ShopID, shop.ShopName)
			}
			return nil
		},
	}
	c.Flags().BoolVar(&idOnly, "id-only", false, "print only the numeric shop ID")
	return c
}

func userIDFromToken(token string) (string, error) {
	userID, _, ok := strings.Cut(token, ".")
	if !ok || userID == "" {
		return "", fmt.Errorf("invalid Etsy access token: expected user_id.token format")
	}
	if _, err := strconv.ParseInt(userID, 10, 64); err != nil {
		return "", fmt.Errorf("invalid Etsy access token user ID: %w", err)
	}
	return userID, nil
}
