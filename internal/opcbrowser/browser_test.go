package opcbrowser

import (
	"reflect"
	"testing"
)

func TestSplitLiteralPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		filter       string
		wantPrefix   []string
		wantWildcard bool
	}{
		{name: "exact path", filter: "/Tag Providers/A/AllData", wantPrefix: []string{"Tag Providers", "A", "AllData"}, wantWildcard: false},
		{name: "wildcard suffix", filter: "/Tag Providers/A/*", wantPrefix: []string{"Tag Providers", "A"}, wantWildcard: true},
		{name: "recursive wildcard", filter: "/**/AllData", wantPrefix: []string{}, wantWildcard: true},
		{name: "root", filter: "/", wantPrefix: nil, wantWildcard: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gotPrefix, gotWildcard := splitLiteralPrefix(test.filter)
			if !reflect.DeepEqual(gotPrefix, test.wantPrefix) {
				t.Fatalf("splitLiteralPrefix(%q) prefix = %#v, want %#v", test.filter, gotPrefix, test.wantPrefix)
			}
			if gotWildcard != test.wantWildcard {
				t.Fatalf("splitLiteralPrefix(%q) wildcard = %v, want %v", test.filter, gotWildcard, test.wantWildcard)
			}
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "adds opc prefix", input: "localhost:4840", want: "opc.tcp://localhost:4840"},
		{name: "keeps prefixed endpoint", input: "opc.tcp://localhost:4840", want: "opc.tcp://localhost:4840"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := normalizeEndpoint(test.input)
			if got != test.want {
				t.Fatalf("normalizeEndpoint(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
