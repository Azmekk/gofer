package executor

import (
	"testing"
)

func TestResolveTemplate(t *testing.T) {
	tests := []struct {
		name    string
		cmd     string
		params  map[string]string
		want    string
		wantErr bool
	}{
		{
			name:   "simple substitution",
			cmd:    "echo {{.name}}",
			params: map[string]string{"name": "world"},
			want:   "echo world",
		},
		{
			name:   "multiple params",
			cmd:    "echo {{.first}} {{.last}}",
			params: map[string]string{"first": "John", "last": "Doe"},
			want:   "echo John Doe",
		},
		{
			name:   "no params needed",
			cmd:    "echo hello",
			params: map[string]string{},
			want:   "echo hello",
		},
		{
			name:    "missing param",
			cmd:     "echo {{.missing}}",
			params:  map[string]string{},
			wantErr: true,
		},
		{
			name:    "template syntax error",
			cmd:     "echo {{.bad",
			params:  map[string]string{},
			wantErr: true,
		},
		{
			name:   "empty command",
			cmd:    "",
			params: map[string]string{},
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveTemplate(tt.cmd, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
