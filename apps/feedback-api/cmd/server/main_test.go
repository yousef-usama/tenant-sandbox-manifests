package main

import "testing"

func TestGetenvFallback(t *testing.T) {
	t.Setenv("TENANT_TEST_VALUE", "")
	if got := getenv("TENANT_TEST_VALUE", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
}

func TestGetenvValue(t *testing.T) {
	t.Setenv("TENANT_TEST_VALUE", "tenant-a")
	if got := getenv("TENANT_TEST_VALUE", "fallback"); got != "tenant-a" {
		t.Fatalf("expected env value, got %q", got)
	}
}
