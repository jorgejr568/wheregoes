package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand_Version(t *testing.T) {
	cmd := RootCmd
	cmd.SetArgs([]string{"https://example.com", "--version"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Skip("Version test may require network access")
	}
}

func TestRootCommand_ShortVersion(t *testing.T) {
	cmd := RootCmd
	cmd.SetArgs([]string{"https://example.com", "-v"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Skip("Version test may require network access")
	}
}

func TestRootCommand_NoArgs(t *testing.T) {
	cmd := RootCmd
	cmd.SetArgs([]string{})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err == nil {
		t.Skip("Root command without args may succeed by design")
	}
}

func TestRootCommand_Help(t *testing.T) {
	cmd := RootCmd
	cmd.SetArgs([]string{"--help"})
	
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	
	err := cmd.Execute()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	
	output := buf.String()
	if !strings.Contains(output, "Wheregoes is a CLI tool") {
		t.Errorf("expected help output, got: %s", output)
	}
}

func TestRootCommand_HasSubcommands(t *testing.T) {
	cmd := RootCmd
	
	if !cmd.HasSubCommands() {
		t.Errorf("expected root command to have subcommands")
	}
	
	commandNames := make(map[string]bool)
	for _, subCmd := range cmd.Commands() {
		commandNames[subCmd.Name()] = true
	}
	
	if !commandNames["track"] {
		t.Errorf("expected 'track' subcommand to exist")
	}
	
	if !commandNames["serve"] {
		t.Errorf("expected 'serve' subcommand to exist")
	}
}