package cmd

import (
	"reflect"
	"testing"
)

func TestParseCORSOrigins(t *testing.T) {
	got := parseCORSOrigins("https://a.com, https://b.com")
	want := []string{"https://a.com", "https://b.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestParseCORSOriginsEmpty(t *testing.T) {
	got := parseCORSOrigins("  ,  ")
	if len(got) != 1 || got[0] != "*" {
		t.Fatalf("want [*], got %v", got)
	}
}
