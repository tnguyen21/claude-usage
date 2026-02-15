package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func loadToken() (string, string, error) {
	// env var override first
	if tok := os.Getenv("CLAUDE_OAUTH_TOKEN"); tok != "" {
		return tok, "", nil
	}

	out, err := exec.Command(
		"security", "find-generic-password",
		"-s", "Claude Code-credentials",
		"-w",
	).Output()
	if err != nil {
		return "", "", fmt.Errorf("no Claude Code credentials found in Keychain")
	}

	var creds KeychainCredentials
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(out))), &creds); err != nil {
		return "", "", fmt.Errorf("failed to parse Keychain credentials: %w", err)
	}

	if creds.ClaudeAiOauth == nil || creds.ClaudeAiOauth.AccessToken == "" {
		return "", "", fmt.Errorf("no OAuth token in Keychain credentials")
	}

	return creds.ClaudeAiOauth.AccessToken, creds.ClaudeAiOauth.SubscriptionType, nil
}
