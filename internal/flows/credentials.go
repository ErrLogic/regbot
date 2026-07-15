package flows

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Character classes used to build compliant passwords.
const (
	lowerChars  = "abcdefghijkmnpqrstuvwxyz"
	upperChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ"
	digitChars  = "23456789"
	symbolChars = "!@#$%^&*()-_=+"
	minPassword = 8
)

// firstNames and lastNames seed generated display names.
var (
	firstNames = []string{"Alex", "Jordan", "Taylor", "Morgan", "Casey", "Riley", "Jamie", "Quinn", "Avery", "Rowan"}
	lastNames  = []string{"Rivera", "Bennett", "Fisher", "Harper", "Nguyen", "Okafor", "Silva", "Turner", "Walsh", "Yamada"}
)

// randInt returns a uniform random int in [0,n) using crypto/rand.
func randInt(n int) (int, error) {
	if n <= 0 {
		return 0, fmt.Errorf("randInt: n must be positive, got %d", n)
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0, fmt.Errorf("randInt: %w", err)
	}
	return int(v.Int64()), nil
}

// randChar returns a random character from set.
func randChar(set string) (byte, error) {
	i, err := randInt(len(set))
	if err != nil {
		return 0, err
	}
	return set[i], nil
}

// GeneratePassword returns a random password of the given length that contains
// at least one lower-case, upper-case, digit, and symbol character. The result
// must never be logged.
func GeneratePassword(length int) (string, error) {
	if length < minPassword {
		length = minPassword
	}
	classes := []string{lowerChars, upperChars, digitChars, symbolChars}
	all := lowerChars + upperChars + digitChars + symbolChars

	buf := make([]byte, 0, length)
	// Guarantee one from each class.
	for _, c := range classes {
		ch, err := randChar(c)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch)
	}
	// Fill the remainder from the combined set.
	for len(buf) < length {
		ch, err := randChar(all)
		if err != nil {
			return "", err
		}
		buf = append(buf, ch)
	}
	// Shuffle so the guaranteed characters are not always at the front.
	if err := shuffle(buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// shuffle performs a Fisher-Yates shuffle using crypto/rand.
func shuffle(b []byte) error {
	for i := len(b) - 1; i > 0; i-- {
		j, err := randInt(i + 1)
		if err != nil {
			return err
		}
		b[i], b[j] = b[j], b[i]
	}
	return nil
}

// GenerateUsername returns prefix followed by a random alphanumeric suffix.
func GenerateUsername(prefix string) (string, error) {
	const suffixLen = 8
	const alphabet = "abcdefghijkmnpqrstuvwxyz23456789"
	var sb strings.Builder
	sb.WriteString(prefix)
	sb.WriteByte('_')
	for i := 0; i < suffixLen; i++ {
		ch, err := randChar(alphabet)
		if err != nil {
			return "", err
		}
		sb.WriteByte(ch)
	}
	return sb.String(), nil
}

// UniqueUsername generates usernames until taken reports false or maxTries is
// reached. taken is a caller-supplied check (e.g. detecting a "username taken"
// screen).
func UniqueUsername(prefix string, taken func(string) (bool, error), maxTries int) (string, error) {
	if maxTries < 1 {
		maxTries = 1
	}
	var lastErr error
	for i := 0; i < maxTries; i++ {
		name, err := GenerateUsername(prefix)
		if err != nil {
			return "", err
		}
		isTaken, err := taken(name)
		if err != nil {
			lastErr = err
			continue
		}
		if !isTaken {
			return name, nil
		}
	}
	if lastErr != nil {
		return "", fmt.Errorf("unique username: exhausted %d tries: %w", maxTries, lastErr)
	}
	return "", fmt.Errorf("unique username: no available name after %d tries", maxTries)
}

// GenerateFullName returns a random "First Last" display name.
func GenerateFullName() (string, error) {
	fi, err := randInt(len(firstNames))
	if err != nil {
		return "", err
	}
	li, err := randInt(len(lastNames))
	if err != nil {
		return "", err
	}
	return firstNames[fi] + " " + lastNames[li], nil
}

// AdultBirthday returns a birthday guaranteeing an age of at least 18 (between
// roughly 20 and 30 years ago) to pass age gates.
func AdultBirthday() (time.Time, error) {
	yearsAgo, err := randInt(11) // 0..10
	if err != nil {
		return time.Time{}, err
	}
	month, err := randInt(12)
	if err != nil {
		return time.Time{}, err
	}
	day, err := randInt(28)
	if err != nil {
		return time.Time{}, err
	}
	year := time.Now().Year() - 20 - yearsAgo
	return time.Date(year, time.Month(month+1), day+1, 0, 0, 0, 0, time.UTC), nil
}
