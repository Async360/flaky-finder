package flaky

import (
	"reflect"
	"testing"
)

func TestGenerateEnv(t *testing.T) {
	tests := []struct {
		name     string
		seedEnvs []SeedEnv
		runIndex int
		want     map[string]string
	}{
		{
			name:     "no seed envs",
			seedEnvs: nil,
			runIndex: 0,
			want:     map[string]string{},
		},
		{
			name:     "single var, first run picks first value",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: []string{"a", "b", "c"}}},
			runIndex: 0,
			want:     map[string]string{"FOO": "a"},
		},
		{
			name:     "single var cycles by run index",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: []string{"a", "b", "c"}}},
			runIndex: 1,
			want:     map[string]string{"FOO": "b"},
		},
		{
			name:     "single var wraps around after the list length",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: []string{"a", "b", "c"}}},
			runIndex: 3,
			want:     map[string]string{"FOO": "a"},
		},
		{
			name:     "single var wraps around, later index",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: []string{"a", "b", "c"}}},
			runIndex: 5,
			want:     map[string]string{"FOO": "c"},
		},
		{
			name: "multiple vars cycle independently",
			seedEnvs: []SeedEnv{
				{Key: "FOO", Values: []string{"a", "b"}},
				{Key: "BAR", Values: []string{"x", "y", "z"}},
			},
			runIndex: 2,
			want:     map[string]string{"FOO": "a", "BAR": "z"},
		},
		{
			name:     "empty values list is skipped",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: nil}},
			runIndex: 0,
			want:     map[string]string{},
		},
		{
			name:     "single value list always picks that value",
			seedEnvs: []SeedEnv{{Key: "FOO", Values: []string{"only"}}},
			runIndex: 42,
			want:     map[string]string{"FOO": "only"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateEnv(tt.seedEnvs, tt.runIndex)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateEnv(%+v, %d) = %v, want %v", tt.seedEnvs, tt.runIndex, got, tt.want)
			}
		})
	}
}

func TestGenerateEnvIsDeterministic(t *testing.T) {
	seedEnvs := []SeedEnv{
		{Key: "FOO", Values: []string{"a", "b", "c"}},
		{Key: "BAR", Values: []string{"1", "2"}},
	}

	for run := 0; run < 20; run++ {
		first := GenerateEnv(seedEnvs, run)
		for i := 0; i < 5; i++ {
			again := GenerateEnv(seedEnvs, run)
			if !reflect.DeepEqual(first, again) {
				t.Fatalf("GenerateEnv is not deterministic at run %d: %v != %v", run, first, again)
			}
		}
	}
}

func TestGenerateEnvFullCycleMatchesInputOrder(t *testing.T) {
	values := []string{"red", "green", "blue"}
	seedEnvs := []SeedEnv{{Key: "COLOR", Values: values}}

	for i, want := range values {
		got := GenerateEnv(seedEnvs, i)
		if got["COLOR"] != want {
			t.Errorf("run %d: got COLOR=%s, want %s", i, got["COLOR"], want)
		}
	}
	// And it should wrap back to the start.
	got := GenerateEnv(seedEnvs, len(values))
	if got["COLOR"] != values[0] {
		t.Errorf("run %d (wrap): got COLOR=%s, want %s", len(values), got["COLOR"], values[0])
	}
}
