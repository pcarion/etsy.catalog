package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrintTokenDoesNotRevealSecrets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	oldOut, oldErr := opts.out, opts.err
	opts.out, opts.err = &stdout, &stderr
	t.Cleanup(func() { opts.out, opts.err = oldOut, oldErr })
	t.Setenv("ETSY_ACCESS_TOKEN", "old-environment-token")
	printToken(tokenResponse{AccessToken: "new-access-secret", RefreshToken: "new-refresh-secret", ExpiresIn: 3600})
	combined := stdout.String() + stderr.String()
	if strings.Contains(combined, "new-access-secret") || strings.Contains(combined, "new-refresh-secret") {
		t.Fatal("token secret was printed")
	}
	if !strings.Contains(combined, "override saved credentials") {
		t.Fatalf("missing override warning: %q", combined)
	}
}
