package cleve

import (
	"slices"
	"testing"
)

func TestApiKey(t *testing.T) {
	plainKey := NewPlainKey()
	apiKey, err := NewAPIKey(plainKey, "user1")
	if err != nil {
		t.Fatal(err.Error())
	}
	if err := apiKey.Compare(plainKey); err != nil {
		t.Error("keys should match, but they don't")
	}
	if slices.Compare(plainKey.Id(), apiKey.Id) != 0 {
		t.Error("key IDs are mismatching")
	}
	newPlainKey := NewPlainKey()
	if err := apiKey.Compare(newPlainKey); err == nil {
		t.Error("keys should not match, but they do")
	}
}
