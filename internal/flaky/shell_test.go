package flaky

import "testing"

func TestJoinShellCommand(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"empty", []string{}, ""},
		{"single simple arg", []string{"echo"}, "'echo'"},
		{"multiple simple args", []string{"npm", "test"}, "'npm' 'test'"},
		{"arg containing spaces stays one token", []string{"echo", "hello world"}, "'echo' 'hello world'"},
		{"arg containing single quote is escaped", []string{"echo", "it's here"}, `'echo' 'it'\''s here'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JoinShellCommand(tt.args)
			if got != tt.want {
				t.Errorf("JoinShellCommand(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
