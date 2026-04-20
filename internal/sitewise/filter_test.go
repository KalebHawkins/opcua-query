package sitewise

import "testing"

func TestNormalizeRootPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty", input: "   ", wantErr: true},
		{name: "adds leading slash", input: "Tag Providers/Line1", want: "/Tag Providers/Line1"},
		{name: "collapses separators", input: `\\Tag Providers\\Line1\\`, want: "/Tag Providers/Line1"},
		{name: "root preserved", input: "/", want: "/"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeRootPath(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("NormalizeRootPath() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("NormalizeRootPath() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestBuildPayload(t *testing.T) {
	t.Parallel()

	payload, err := BuildPayload("/**/PLC*")
	if err != nil {
		t.Fatalf("BuildPayload() error = %v", err)
	}

	if payload.RootPath != "/**/PLC*" {
		t.Fatalf("RootPath = %q, want %q", payload.RootPath, "/**/PLC*")
	}

	if payload.JSON == "" {
		t.Fatal("BuildPayload() returned empty JSON")
	}
}

func TestMatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		rule      string
		candidate string
		want      bool
	}{
		{name: "match exact", rule: "/Tag Providers/A/AllData", candidate: "/Tag Providers/A/AllData", want: true},
		{name: "single segment wildcard", rule: "/Tag Providers/*/AllData", candidate: "/Tag Providers/A/AllData", want: true},
		{name: "recursive wildcard", rule: "/**/AllData", candidate: "/Tag Providers/A/AllData", want: true},
		{name: "no match", rule: "/**/Counter*", candidate: "/Tag Providers/A/AllData", want: false},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Match(test.rule, test.candidate)
			if got != test.want {
				t.Fatalf("Match(%q, %q) = %v, want %v", test.rule, test.candidate, got, test.want)
			}
		})
	}
}
