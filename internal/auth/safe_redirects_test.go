package auth

import (
	"testing"
)

func TestSafeRedirects(t *testing.T) {
	cases := []struct {
		name   string
		url    string
		expect string
	}{
		{
			name:   "absolute url",
			url:    "https://google.com",
			expect: defaultRedirect,
		},
		{
			name:   "protocol-relative",
			url:    `\\evil`,
			expect: defaultRedirect,
		},
		{
			name:   "safe relative url",
			url:    "/runs",
			expect: "/runs",
		},
		{
			name:   "safe relative url",
			url:    "/runs/this-is-a-run-id",
			expect: "/runs/this-is-a-run-id",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			safeURL := SafeRedirectURL(c.url)
			if safeURL != c.expect {
				t.Errorf("expected url %s, got %s", c.expect, safeURL)
			}
		})
	}
}
