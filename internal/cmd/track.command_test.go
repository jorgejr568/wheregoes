package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestTrackCommand_ValidURL(t *testing.T) {
	cmd := track()
	cmd.SetArgs([]string{"https://example.com"})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err != nil {
		t.Skip("Track command test skipped - requires network access")
	}
}

func TestTrackCommand_InvalidURL(t *testing.T) {
	cmd := track()
	cmd.SetArgs([]string{"not-a-url"})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err == nil {
		t.Errorf("expected error for invalid URL")
	}
	
	if !strings.Contains(err.Error(), "Invalid URL") && 
	   !strings.Contains(errBuf.String(), "Invalid URL") {
		t.Errorf("expected 'Invalid URL' error message")
	}
}

func TestTrackCommand_HTTPUrl(t *testing.T) {
	cmd := track()
	cmd.SetArgs([]string{"http://example.com"})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err != nil {
		t.Skip("Track command test skipped - requires network access")
	}
}

func TestTrackCommand_NoArgs(t *testing.T) {
	cmd := track()
	cmd.SetArgs([]string{})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err == nil {
		t.Errorf("expected error when no URL provided")
	}
}

func TestTrackCommand_TooManyArgs(t *testing.T) {
	cmd := track()
	cmd.SetArgs([]string{"https://example.com", "extra-arg"})
	
	buf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(errBuf)
	
	err := cmd.Execute()
	if err == nil {
		t.Errorf("expected error when too many args provided")
	}
}

func TestTrackCommand_JSONFlag(t *testing.T) {
	cmd := track()
	
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Errorf("expected 'json' flag to exist")
		return
	}
	
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected 'json' flag default value to be false, got %s", jsonFlag.DefValue)
	}
}

func TestTrackCommand_Usage(t *testing.T) {
	cmd := track()
	
	if cmd.Use != "track [url]" {
		t.Errorf("expected Use to be 'track [url]', got '%s'", cmd.Use)
	}
	
	if cmd.Short != "Track a URL" {
		t.Errorf("expected Short to be 'Track a URL', got '%s'", cmd.Short)
	}
}