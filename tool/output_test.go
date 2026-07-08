package tool

import "testing"

func TestResolveOutputBaseName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"", "main", false},
		{"video", "video", false},
		{"video.ts", "video", false},
		{"video.mp4", "video", false},
		{"my.movie.ts", "my.movie", false},
		{"../evil.mp4", "", true},
		{"path/video.mp4", "", true},
		{".mp4", "", true},
	}

	for _, tt := range tests {
		got, err := ResolveOutputBaseName(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Fatalf("expected error for %q", tt.input)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", tt.input, err)
		}
		if got != tt.expected {
			t.Fatalf("ResolveOutputBaseName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
