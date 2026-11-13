package auth

import (
	"strings"
)

const defaultRedirect = "/"

var unsafeRunes = []rune{'\r', '\t', '\n', '\\'}

func allSafeRunes(url string) bool {
	for _, r := range unsafeRunes {
		if strings.ContainsRune(url, r) {
			return false
		}
	}
	return true
}

func SafeRedirectURL(url string) string {
	if !strings.HasPrefix(url, "/") {
		return defaultRedirect
	}
	if strings.HasPrefix(url, "//") {
		return defaultRedirect
	}
	if !allSafeRunes(url) {
		return defaultRedirect
	}
	return url
}
