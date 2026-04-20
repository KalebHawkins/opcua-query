package opcbrowser

import "testing"

func TestMatchesFindQuery(t *testing.T) {
	t.Parallel()

	match := Match{
		Path:        "/Plant/Area 1/Counter",
		BrowseName:  "Counter01",
		DisplayName: "Line Counter",
		NodeID:      "ns=2;s=Counter01",
	}

	tests := []struct {
		name  string
		query string
		want  bool
	}{
		{name: "matches path", query: "area 1", want: true},
		{name: "matches browse name", query: "counter01", want: true},
		{name: "matches display name", query: "line counter", want: true},
		{name: "matches node id", query: "ns=2", want: true},
		{name: "no match", query: "temperature", want: false},
		{name: "blank", query: "   ", want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := matchesFindQuery(test.query, match)
			if got != test.want {
				t.Fatalf("matchesFindQuery(%q) = %v, want %v", test.query, got, test.want)
			}
		})
	}
}

func TestNormalizeBrowsePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty becomes root", input: "", want: "/"},
		{name: "keeps root", input: "/", want: "/"},
		{name: "adds leading slash", input: "Plant/Area 1", want: "/Plant/Area 1"},
		{name: "trims trailing slash", input: "/Plant/Area 1/", want: "/Plant/Area 1"},
		{name: "rejects wildcard", input: "/Plant/*", wantErr: true},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := normalizeBrowsePath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("normalizeBrowsePath(%q) error = nil, want error", test.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeBrowsePath(%q) error = %v", test.input, err)
			}
			if got != test.want {
				t.Fatalf("normalizeBrowsePath(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
