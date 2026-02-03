package schema

import (
	"fmt"
	"testing"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		wantErrs  int // expected number of errors; 0 means valid
		wantMatch string // if non-empty, at least one error must contain this substring
	}{
		{
			name:     "valid minimal",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo"}]}}}`,
			wantErrs: 0,
		},
		{
			name:      "missing tasks",
			json:      `{}`,
			wantErrs:  1,
			wantMatch: "missing required field: tasks",
		},
		{
			name:      "missing desc",
			json:      `{"tasks":{"t":{"steps":[{"cmd":"echo"}]}}}`,
			wantErrs:  1,
			wantMatch: "missing required field: desc",
		},
		{
			name:      "missing steps",
			json:      `{"tasks":{"t":{"desc":"d"}}}`,
			wantErrs:  1,
			wantMatch: "missing required field: steps",
		},
		{
			name:      "step with no type",
			json:      `{"tasks":{"t":{"desc":"d","steps":[{}]}}}`,
			wantErrs:  1,
			wantMatch: "must have exactly one of cmd, ref, or concurrent",
		},
		{
			name:      "step with multiple types",
			json:      `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","ref":"x"}]}}}`,
			wantErrs:  1,
			wantMatch: "must have exactly one of cmd, ref, or concurrent",
		},
		{
			name:      "invalid os value",
			json:      `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","os":"freebsd"}]}}}`,
			wantErrs:  1,
			wantMatch: "invalid os value",
		},
		{
			name:     "valid os linux",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","os":"linux"}]}}}`,
			wantErrs: 0,
		},
		{
			name:     "valid os darwin",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","os":"darwin"}]}}}`,
			wantErrs: 0,
		},
		{
			name:     "valid os windows",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","os":"windows"}]}}}`,
			wantErrs: 0,
		},
		{
			name:     "valid os wildcard",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"cmd":"echo","os":"*"}]}}}`,
			wantErrs: 0,
		},
		{
			name:      "duplicate task keys",
			json:      `{"tasks":{"t":{"desc":"a","steps":[{"cmd":"echo"}]},"t":{"desc":"b","steps":[{"cmd":"echo"}]}}}`,
			wantErrs:  1,
			wantMatch: "duplicate task name",
		},
		{
			name:     "valid concurrent steps",
			json:     `{"tasks":{"t":{"desc":"d","steps":[{"concurrent":[{"cmd":"echo a"},{"cmd":"echo b"}]}]}}}`,
			wantErrs: 0,
		},
		{
			name:      "invalid concurrent sub-step",
			json:      `{"tasks":{"t":{"desc":"d","steps":[{"concurrent":[{}]}]}}}`,
			wantErrs:  1,
			wantMatch: "must have exactly one of cmd, ref, or concurrent",
		},
		{
			name:      "param missing name",
			json:      `{"tasks":{"t":{"desc":"d","params":[{}],"steps":[{"cmd":"echo"}]}}}`,
			wantErrs:  1,
			wantMatch: "missing required field: name",
		},
		{
			name:     "param with name",
			json:     `{"tasks":{"t":{"desc":"d","params":[{"name":"x"}],"steps":[{"cmd":"echo"}]}}}`,
			wantErrs: 0,
		},
		{
			name:      "invalid json",
			json:      `{not json}`,
			wantErrs:  1,
			wantMatch: "invalid JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := Validate([]byte(tt.json))
			if len(errs) != tt.wantErrs {
				t.Errorf("got %d errors, want %d: %v", len(errs), tt.wantErrs, errs)
				return
			}
			if tt.wantMatch != "" && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if contains(err.Error(), tt.wantMatch) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no error contains %q; got: %v", tt.wantMatch, errs)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestValidate_MultipleTaskErrors(t *testing.T) {
	// Two tasks both missing desc
	json := `{"tasks":{"a":{"steps":[{"cmd":"echo"}]},"b":{"steps":[{"cmd":"echo"}]}}}`
	errs := Validate([]byte(json))
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), errs)
	}
	for _, err := range errs {
		fmt.Println(err) // debug visibility
	}
}
