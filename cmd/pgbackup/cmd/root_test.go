package cmd

import (
	"testing"
)

func TestRootNoArgs(t *testing.T) {
	rootCmd.SetArgs([]string{})
	err := rootCmd.Execute()
	if err != nil {
		t.Errorf("expected rootCmd.Execute() with no args to return nil, got: %v", err)
	}
}

func TestSubcommandsRecognized(t *testing.T) {
	subcommands := map[string]bool{
		"backup":  false,
		"restore": false,
		"diff":    false,
	}
	for _, sub := range rootCmd.Commands() {
		if _, ok := subcommands[sub.Name()]; ok {
			subcommands[sub.Name()] = true
		}
	}
	for name, found := range subcommands {
		if !found {
			t.Errorf("expected subcommand %q to be registered, but it was not found", name)
		}
	}
}
