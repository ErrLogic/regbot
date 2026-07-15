package flows

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func hasAny(s, set string) bool { return strings.ContainsAny(s, set) }

func TestGeneratePasswordPolicy(t *testing.T) {
	for _, length := range []int{8, 16, 32} {
		pw, err := GeneratePassword(length)
		if err != nil {
			t.Fatalf("GeneratePassword(%d): %v", length, err)
		}
		if len(pw) != length {
			t.Errorf("len = %d, want %d", len(pw), length)
		}
		if !hasAny(pw, lowerChars) || !hasAny(pw, upperChars) ||
			!hasAny(pw, digitChars) || !hasAny(pw, symbolChars) {
			t.Errorf("password %q missing a required character class", pw)
		}
	}
}

func TestGeneratePasswordClampsShortLength(t *testing.T) {
	pw, err := GeneratePassword(4)
	if err != nil {
		t.Fatalf("GeneratePassword: %v", err)
	}
	if len(pw) < minPassword {
		t.Errorf("short length not clamped: len=%d", len(pw))
	}
}

func TestGeneratePasswordUnique(t *testing.T) {
	a, _ := GeneratePassword(16)
	b, _ := GeneratePassword(16)
	if a == b {
		t.Error("two generated passwords should differ")
	}
}

func TestGenerateUsername(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		u, err := GenerateUsername("user")
		if err != nil {
			t.Fatalf("GenerateUsername: %v", err)
		}
		if !strings.HasPrefix(u, "user_") {
			t.Errorf("username %q missing prefix", u)
		}
		if seen[u] {
			t.Errorf("duplicate username generated: %q", u)
		}
		seen[u] = true
	}
}

func TestUniqueUsernameRegenerates(t *testing.T) {
	calls := 0
	name, err := UniqueUsername("user", func(string) (bool, error) {
		calls++
		return calls < 3, nil // taken on first two tries
	}, 5)
	if err != nil {
		t.Fatalf("UniqueUsername: %v", err)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
	if !strings.HasPrefix(name, "user_") {
		t.Errorf("name = %q", name)
	}
}

func TestUniqueUsernameExhausted(t *testing.T) {
	_, err := UniqueUsername("user", func(string) (bool, error) { return true, nil }, 3)
	if err == nil {
		t.Fatal("expected exhaustion error")
	}
}

func TestUniqueUsernamePropagatesCheckError(t *testing.T) {
	boom := errors.New("check failed")
	_, err := UniqueUsername("user", func(string) (bool, error) { return false, boom }, 2)
	if err == nil || !errors.Is(err, boom) {
		t.Fatalf("want wrapped check error, got %v", err)
	}
}

func TestAdultBirthday(t *testing.T) {
	for i := 0; i < 20; i++ {
		bd, err := AdultBirthday()
		if err != nil {
			t.Fatalf("AdultBirthday: %v", err)
		}
		age := time.Since(bd).Hours() / 24 / 365.25
		if age < 18 {
			t.Errorf("birthday %v yields age %.1f < 18", bd, age)
		}
	}
}

func TestGenerateFullName(t *testing.T) {
	name, err := GenerateFullName()
	if err != nil {
		t.Fatalf("GenerateFullName: %v", err)
	}
	if !strings.Contains(name, " ") {
		t.Errorf("full name %q should contain a space", name)
	}
}
