package engine

import "testing"

func TestCitationFabricationSentinel_DetectsFabricatedURLs(t *testing.T) {
	sentinel := NewCitationFabricationSentinel([]string{"docs.example.org"})
	text := "See https://docs.example.org and https://fake-hallucinated-site.xyz for details."

	flagged, fabricated := sentinel.Inspect(text, nil)
	if !flagged {
		t.Fatal("expected fabricated URL to be flagged")
	}
	if len(fabricated) != 1 || fabricated[0] != "https://fake-hallucinated-site.xyz" {
		t.Fatalf("unexpected fabricated list: %v", fabricated)
	}
}
