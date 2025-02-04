package str

import "testing"

func Test_ReplaceDashWithCamelCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"monaco-f1", "MonacoF1"},
		{"monaco-game", "MonacoGame"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ToReplaceDashWithCamelCase(tt.input); got != tt.want {
				t.Errorf("toReplaceDashWithCamelCase(\"%v\") = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
