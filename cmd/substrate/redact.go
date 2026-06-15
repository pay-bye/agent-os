package main

import (
	"net/url"
	"regexp"
	"strings"
)

func redact(message string, secrets []string) string {
	redacted := redactDatabaseURLs(message)
	for _, secret := range secrets {
		if secret == "" {
			continue
		}
		redacted = strings.ReplaceAll(redacted, secret, "<redacted>")
		redacted = strings.ReplaceAll(redacted, redactURL(secret), "<redacted>")
	}
	return redacted
}

func redactDatabaseURLs(message string) string {
	pattern := regexp.MustCompile(`postgres://[^\s]+`)
	return pattern.ReplaceAllStringFunc(message, redactURL)
}

func redactURL(value string) string {
	parsed, err := url.Parse(value)
	if err != nil || parsed.User == nil {
		return value
	}
	if _, ok := parsed.User.Password(); !ok {
		return value
	}
	parsed.User = url.UserPassword(parsed.User.Username(), "<redacted>")
	return parsed.String()
}
