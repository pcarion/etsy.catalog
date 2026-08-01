package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var state string
	var limit int
	c := &cobra.Command{Use: "list", Short: "List the shop's listings", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if err := requireShop(); err != nil {
			return err
		}
		cl, err := client()
		if err != nil {
			return err
		}
		q := url.Values{"limit": {strconv.Itoa(limit)}}
		if state != "" {
			q.Set("state", state)
		}
		var response struct {
			Results []struct {
				ListingID int64  `json:"listing_id"`
				State     string `json:"state"`
				Title     string `json:"title"`
			} `json:"results"`
			Count int `json:"count"`
		}
		if err := cl.Do(cmd.Context(), http.MethodGet, fmt.Sprintf("shops/%s/listings", opts.shopID), q, nil, &response); err != nil {
			return err
		}
		fmt.Fprintln(opts.out, "ID\tSTATE\tTITLE")
		for _, l := range response.Results {
			fmt.Fprintf(opts.out, "%d\t%s\t%s\n", l.ListingID, l.State, l.Title)
		}
		return nil
	}}
	c.Flags().StringVar(&state, "state", "active", "listing state (active, inactive, draft, expired, sold_out)")
	c.Flags().IntVar(&limit, "limit", 100, "maximum listings to return (1-100)")
	return c
}
