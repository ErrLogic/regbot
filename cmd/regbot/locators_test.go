package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVerifyLocatorsShippedFiles(t *testing.T) {
	var buf bytes.Buffer
	problems := verifyLocators(&buf, filepath.Join("..", "..", "locators"))
	if problems != 0 {
		t.Fatalf("shipped locators have %d problems:\n%s", problems, buf.String())
	}
	out := buf.String()
	for _, app := range []string{"instagram", "tiktok", "gmail"} {
		if !strings.Contains(out, app) {
			t.Errorf("summary missing %s:\n%s", app, out)
		}
	}
	if !strings.Contains(out, "OK") {
		t.Errorf("expected OK lines:\n%s", out)
	}
}

func TestVerifyLocatorsMissingDir(t *testing.T) {
	var buf bytes.Buffer
	// Empty temp dir: no files -> every check fails to load.
	problems := verifyLocators(&buf, t.TempDir())
	if problems != len(locatorChecks) {
		t.Fatalf("problems = %d, want %d\n%s", problems, len(locatorChecks), buf.String())
	}
}

func TestVerifyLocatorsMissingElement(t *testing.T) {
	dir := t.TempDir()
	// A valid instagram file missing a required element; gmail/tiktok absent.
	body := `{"version":"x","elements":{"email_field":[{"by":"id","selector":"id/x"}]}}`
	if err := os.WriteFile(filepath.Join(dir, "instagram.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	problems := verifyLocators(&buf, dir)
	if problems == 0 {
		t.Fatal("expected problems for missing required elements")
	}
	if !strings.Contains(buf.String(), "FAIL") {
		t.Errorf("expected FAIL in output:\n%s", buf.String())
	}
}
