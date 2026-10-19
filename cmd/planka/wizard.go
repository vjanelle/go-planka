package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	huh "charm.land/huh/v2"
	"github.com/choria-io/fisk"
)

// validateInstanceURL validates a Planka instance URL and returns its
// normalized form. It requires an absolute http(s) URL and strips a
// single trailing slash.
func validateInstanceURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("instance URL is required")
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %v", s, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("%q is not a valid http(s) instance URL", s)
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

// nonEmpty rejects blank values for required form fields.
func nonEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("required")
	}
	return nil
}

// helpRequested reports whether the user explicitly asked for help.
// fisk prints usage itself but may still run the command action, so the
// wizard must stand aside in that case.
func helpRequested(args []string) bool {
	for _, a := range args {
		switch a {
		case "--help", "--help=true", "-h":
			return true
		}
	}
	return false
}

// configured reports whether a usable saved configuration exists. A
// missing file or a file with an empty URL counts as unconfigured.
func configured(cfg *Config) bool { return cfg.URL != "" }

// firstMissingLoginArg returns the first empty login argument, in
// declaration order, as the same error fisk emits for a missing required
// argument, or nil when all are provided.
func firstMissingLoginArg(url, username, password string) error {
	if url == "" {
		return fmt.Errorf("%w '%s' not provided", fisk.ErrRequiredArgument, "url")
	}
	if username == "" {
		return fmt.Errorf("%w '%s' not provided", fisk.ErrRequiredArgument, "username")
	}
	if password == "" {
		return fmt.Errorf("%w '%s' not provided", fisk.ErrRequiredArgument, "password")
	}
	return nil
}

// firstMissingAPIKeyArg returns the first empty api-key argument, in
// declaration order, as the same error fisk emits for a missing required
// argument, or nil when all are provided.
func firstMissingAPIKeyArg(url, key string) error {
	if url == "" {
		return fmt.Errorf("%w '%s' not provided", fisk.ErrRequiredArgument, "url")
	}
	if key == "" {
		return fmt.Errorf("%w '%s' not provided", fisk.ErrRequiredArgument, "key")
	}
	return nil
}

// newLoginForm builds the interactive login wizard: instance URL,
// username, and password in a single form.
func newLoginForm(url, username, password *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Planka instance URL *").
				Key("url").
				Value(url).
				Validate(func(s string) error {
					_, err := validateInstanceURL(s)
					return err
				}),
			huh.NewInput().
				Title("Username *").
				Key("username").
				Value(username).
				Validate(nonEmpty),
			huh.NewInput().
				Title("Password *").
				Key("password").
				Value(password).
				EchoMode(huh.EchoModePassword).
				Validate(nonEmpty),
		),
	)
}

// newAPIKeyForm builds the interactive API-key wizard: instance URL and
// API key in a single form.
func newAPIKeyForm(url, apiKey *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Planka instance URL *").
				Key("url").
				Value(url).
				Validate(func(s string) error {
					_, err := validateInstanceURL(s)
					return err
				}),
			huh.NewInput().
				Title("API key *").
				Key("api_key").
				Value(apiKey).
				EchoMode(huh.EchoModePassword).
				Validate(nonEmpty),
		),
	)
}

// promptLogin runs the login wizard, prefilling any empty input from the
// existing configuration. It returns the normalized URL, username, and
// password. On user abort it prints a message and exits with status 1.
func promptLogin(url, username, password string, cfg *Config) (string, string, string, error) {
	if url == "" {
		url = cfg.URL
	}
	if username == "" {
		username = cfg.Username
	}

	form := newLoginForm(&url, &username, &password)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Aborted.")
			os.Exit(1)
		}
		return "", "", "", err
	}

	normalized, err := validateInstanceURL(url)
	if err != nil {
		return "", "", "", err
	}
	return normalized, username, password, nil
}

// promptAPIKey runs the API-key wizard, prefilling the URL from the
// existing configuration when empty. It returns the normalized URL and
// API key. On user abort it prints a message and exits with status 1.
func promptAPIKey(url, apiKey string, cfg *Config) (string, string, error) {
	if url == "" {
		url = cfg.URL
	}

	form := newAPIKeyForm(&url, &apiKey)
	if err := form.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Aborted.")
			os.Exit(1)
		}
		return "", "", err
	}

	normalized, err := validateInstanceURL(url)
	if err != nil {
		return "", "", err
	}
	return normalized, apiKey, nil
}
