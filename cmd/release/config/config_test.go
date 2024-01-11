package config

import (
	"embed"
	"testing"
)

//go:embed test_data/config.json
var configFS embed.FS

func TestRead(t *testing.T) {
	f, err := configFS.Open("test_data/config.json")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := read(f); err != nil {
		t.Fatal(err)
	}
}
