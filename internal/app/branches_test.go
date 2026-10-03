package app

import (
	"reflect"
	"testing"
)

func TestParseRemoteBranches(t *testing.T) {
	output := "65049fb8fccea09cc947147362a92031dffd9bf8\trefs/heads/main\n" +
		"f59319031bf01430d0581d0ad931a9866044f22c\trefs/heads/feat/login\n" +
		"fd51812edfc12a889ebc8023c727bdb1445f8bae\trefs/tags/v1.0.0\n" +
		"\n"
	got := parseRemoteBranches(output)
	want := []string{"main", "feat/login"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseRemoteBranches() = %v, want %v", got, want)
	}
}

func TestParseRemoteBranchesEmpty(t *testing.T) {
	if got := parseRemoteBranches(""); got != nil {
		t.Fatalf("parseRemoteBranches(\"\") = %v, want nil", got)
	}
}

func TestBranchCompletions(t *testing.T) {
	branches := []string{"main", "develop", "feat/login", "feat/logout", "fix/crash"}

	tests := []struct {
		prefix string
		want   []string
	}{
		{"", branches},
		{"feat/", []string{"feat/login", "feat/logout"}},
		{"main", []string{"main"}},
		{"  ", branches},
	}
	for _, tt := range tests {
		got := branchCompletions(branches, tt.prefix)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("branchCompletions(%q) = %v, want %v", tt.prefix, got, tt.want)
		}
	}
	if got := branchCompletions(branches, "nomatch"); len(got) != 0 {
		t.Errorf("branchCompletions(\"nomatch\") = %v, want empty", got)
	}
}
