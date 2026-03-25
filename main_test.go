package main

import (
	"net/http"
	"testing"
)

func TestNewHTTPServerUsesSafeTimeoutDefaults(t *testing.T) {
	t.Parallel()

	config := testConfig(t)
	handler := http.NewServeMux()
	server := newHTTPServer(config, handler)

	if got, want := server.Addr, ":"+config.Port; got != want {
		t.Fatalf("server addr = %q, want %q", got, want)
	}
	if server.Handler != handler {
		t.Fatal("server handler did not match input handler")
	}
	if got := server.ReadHeaderTimeout; got != httpServerReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", got, httpServerReadHeaderTimeout)
	}
	if got := server.ReadTimeout; got != httpServerReadTimeout {
		t.Fatalf("ReadTimeout = %s, want %s", got, httpServerReadTimeout)
	}
	if got := server.IdleTimeout; got != httpServerIdleTimeout {
		t.Fatalf("IdleTimeout = %s, want %s", got, httpServerIdleTimeout)
	}
	if got := server.WriteTimeout; got != 0 {
		t.Fatalf("WriteTimeout = %s, want 0", got)
	}
}
