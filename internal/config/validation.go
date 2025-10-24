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

// ValidateCloudflaredToken checks if the given Cloudflare tunnel token has valid format
func ValidateCloudflaredToken(token string) error {
	slog.Debug("ValidateCloudflaredToken()", "tokenLength", len(token))

	// Empty token is allowed (Cloudflared disabled)
	if token == "" {
		return nil
	}

	// Cloudflare tunnel tokens are JWTs that start with "eyJ"
	if !strings.HasPrefix(token, "eyJ") {
		return fmt.Errorf("invalid Cloudflare tunnel token format: must start with 'eyJ'")
	}

	// Check minimum length (JWT tokens are typically quite long)
	if len(token) < 100 {
		return fmt.Errorf("invalid Cloudflare tunnel token: too short")
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

// ValidateOAuthClientId checks if the OAuth client ID is valid
func ValidateOAuthClientId(id string, oauthEnabled bool) error {
	slog.Debug("ValidateOAuthClientId()", "idLength", len(id), "oauthEnabled", oauthEnabled)

	// If OAuth is disabled, client ID is not required
	if !oauthEnabled {
		return nil
	}

	// If OAuth is enabled, client ID must be non-empty
	if id == "" {
		return fmt.Errorf("OAuth client ID is required when OAuth is enabled")
	}

	// Basic validation - client ID should have reasonable length
	if len(id) < 10 {
		return fmt.Errorf("OAuth client ID is too short")
	}

	return nil
}

// ValidateOAuthClientSecret checks if the OAuth client secret is valid
func ValidateOAuthClientSecret(secret string, oauthEnabled bool) error {
	slog.Debug("ValidateOAuthClientSecret()", "secretLength", len(secret), "oauthEnabled", oauthEnabled)

	// If OAuth is disabled, client secret is not required
	if !oauthEnabled {
		return nil
	}

	// Empty secret is allowed when OAuth is enabled (preserves existing value)
	if secret == "" {
		return nil
	}

	// If provided, secret should have reasonable length for security
	if len(secret) < 20 {
		return fmt.Errorf("OAuth client secret is too short (minimum 20 characters)")
	}

	return nil
}

// ValidateOAuthIssuerUrl checks if the OAuth issuer URL is valid
func ValidateOAuthIssuerUrl(url string, oauthEnabled bool) error {
	slog.Debug("ValidateOAuthIssuerUrl()", "url", url, "oauthEnabled", oauthEnabled)

	// If OAuth is disabled, issuer URL is not required
	if !oauthEnabled {
		return nil
	}

	// If OAuth is enabled, issuer URL must be non-empty
	if url == "" {
		return fmt.Errorf("OAuth issuer URL is required when OAuth is enabled")
	}

	// Basic URL validation - should start with https://
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("OAuth issuer URL must start with 'https://'")
	}

	// Should have reasonable length and structure
	if len(url) < 15 {
		return fmt.Errorf("OAuth issuer URL is too short")
	}

	return nil
}

// ValidatePublicDomain checks if the public domain is valid
func ValidatePublicDomain(domain string, oauthEnabled bool) error {
	slog.Debug("ValidatePublicDomain()", "domain", domain, "oauthEnabled", oauthEnabled)

	// If OAuth is disabled, public domain is not required
	if !oauthEnabled {
		return nil
	}

	// If OAuth is enabled, public domain must be non-empty
	if domain == "" {
		return fmt.Errorf("public domain is required when OAuth is enabled")
	}

	// Domain should not include protocol prefix
	if strings.HasPrefix(domain, "http://") || strings.HasPrefix(domain, "https://") {
		return fmt.Errorf("public domain should not include 'http://' or 'https://' prefix")
	}

	// Basic domain validation - should contain at least one dot
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("public domain must be a valid domain name (e.g., immich.example.com)")
	}

	// Should have reasonable length
	if len(domain) < 4 {
		return fmt.Errorf("public domain is too short")
	}

	return nil
}
