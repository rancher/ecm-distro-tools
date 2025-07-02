package config

import (
	"strings"
	"testing"
)

func TestRead(t *testing.T) {
	conf, err := ExampleConfig()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Read(strings.NewReader(conf))
	if err != nil {
		t.Fatal(err)
	}
}
