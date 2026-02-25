package main

import (
	"net/http"
	"testing"

	"github.com/gnailuy/amiglot-api/internal/config"
)

func TestRunServer_UsesPort(t *testing.T) {
	var gotAddr string

	err := runServer(config.Config{Port: "1234"}, nil, func(addr string, handler http.Handler) error {
		gotAddr = addr
		if handler == nil {
			t.Fatalf("expected handler to be set")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if gotAddr != ":1234" {
		t.Fatalf("expected addr :1234, got %s", gotAddr)
	}
}
