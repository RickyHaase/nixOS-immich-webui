package config

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// ValidateTimezone checks if the given timezone string is a valid IANA timezone
func ValidateTimezone(tz string) error {
	slog.Debug("ValidateTimezone()", "timezone", tz)

	if tz == "" {
		return fmt.Errorf("timezone cannot be empty")
	}

	// Try to load the timezone to verify it's valid
	_, err := time.LoadLocation(tz)
	if err != nil {
		return fmt.Errorf("invalid timezone '%s': %w", tz, err)
	}

	return nil
}

// ValidateTimeFormat checks if the given time string is in valid HH:MM format (00:00-23:59)
func ValidateTimeFormat(t string) error {
	slog.Debug("ValidateTimeFormat()", "time", t)

	if t == "" {
		return fmt.Errorf("time cannot be empty")
	}

	// Parse the time using 24-hour format
	_, err := time.Parse("15:04", t)
	if err != nil {
		return fmt.Errorf("invalid time format '%s': must be HH:MM (00:00-23:59)", t)
	}

	return nil
}

// ValidateTailscaleAuthKey checks if the given Tailscale auth key has valid format
func ValidateTailscaleAuthKey(key string) error {
	slog.Debug("ValidateTailscaleAuthKey()", "keyLength", len(key))

	// Empty key is allowed (Tailscale disabled)
	if key == "" {
		return nil
	}

	// Check if key starts with tskey- prefix
	if !strings.HasPrefix(key, "tskey-") {
		return fmt.Errorf("invalid Tailscale auth key format: must start with 'tskey-'")
	}

	// Check minimum length (tskey- prefix + some key data)
	if len(key) < 20 {
		return fmt.Errorf("invalid Tailscale auth key: too short")
	}

	return nil
}

// ValidateEmail checks if the given email address has a valid format
func ValidateEmail(email string) error {
	slog.Debug("ValidateEmail()", "email", email)

	// Empty email is allowed (email notifications disabled)
	if email == "" {
		return nil
	}

	// Basic email regex pattern
	// This matches: localpart@domain.tld
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

	if !emailRegex.MatchString(email) {
		return fmt.Errorf("invalid email format: '%s'", email)
	}

	return nil
}
