package config

import (
	"errors"
	"log/slog"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Reader reads typed settings from a Source and collects every problem, so a configuration reports all of
// them at once. A setting that is unset, empty or only spaces takes its default. A getter that finds an
// invalid value records the problem under the setting's name and returns the default.
type Reader struct {
	src      Source
	problems []string
}

// NewReader returns a Reader over source.
func NewReader(source Source) *Reader {
	return &Reader{src: source}
}

// String returns the setting key without surrounding spaces, or fallback.
func (r *Reader) String(key, fallback string) string {
	if value, ok := r.lookup(key); ok {
		return value
	}
	return fallback
}

// Required returns the setting key, recording a problem when it is not set.
func (r *Reader) Required(key string) string {
	value, ok := r.lookup(key)
	if !ok {
		r.Fail(key, "is required")
	}
	return value
}

// OneOf returns the setting key when it is one of allowed, or fallback.
func (r *Reader) OneOf(key, fallback string, allowed ...string) string {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	if !slices.Contains(allowed, value) {
		r.Fail(key, "must be one of "+strings.Join(allowed, ", "))
		return fallback
	}
	return value
}

// Duration returns the setting key as a positive duration in Go syntax, such as 15s, or fallback.
func (r *Reader) Duration(key string, fallback time.Duration) time.Duration {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		r.Fail(key, "must be a positive duration such as 15s")
		return fallback
	}
	return parsed
}

// PositiveInt returns the setting key as an integer of 1 or more, or fallback.
func (r *Reader) PositiveInt(key string, fallback int64) int64 {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		r.Fail(key, "must be a positive integer")
		return fallback
	}
	return parsed
}

// Ratio returns the setting key as a number from 0 to 1, or fallback.
func (r *Reader) Ratio(key string, fallback float64) float64 {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || !(parsed >= 0 && parsed <= 1) {
		r.Fail(key, "must be a number from 0 to 1")
		return fallback
	}
	return parsed
}

// URL returns the setting key when it is an http or https URL with a host, and "" when it is unset.
func (r *Reader) URL(key string) string {
	value, ok := r.lookup(key)
	if !ok {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		r.Fail(key, "must be an http or https URL such as http://collector:4318")
		return ""
	}
	return value
}

// Bool returns the setting key as a boolean (true, false, 1, 0 …), or fallback.
func (r *Reader) Bool(key string, fallback bool) bool {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		r.Fail(key, "must be true or false")
		return fallback
	}
	return parsed
}

// List returns the comma-separated setting key without blank entries or surrounding spaces, or nil.
func (r *Reader) List(key string) []string {
	value, ok := r.lookup(key)
	if !ok {
		return nil
	}

	var entries []string
	for part := range strings.SplitSeq(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			entries = append(entries, part)
		}
	}
	return entries
}

// Fail records that the setting key breaks a rule the caller checks, such as one setting exceeding another.
func (r *Reader) Fail(key, reason string) {
	r.problems = append(r.problems, key+": "+reason)
}

// Err returns every recorded problem in one error, as "invalid configuration: KEY: reason; …", or nil.
func (r *Reader) Err() error {
	if len(r.problems) == 0 {
		return nil
	}
	return errors.New("invalid configuration: " + strings.Join(r.problems, "; "))
}

func (r *Reader) lookup(key string) (string, bool) {
	value, ok := r.src(key)
	value = strings.TrimSpace(value)
	return value, ok && value != ""
}

func (r *Reader) level(key string, fallback slog.Level) slog.Level {
	value, ok := r.lookup(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	}
	r.Fail(key, "must be one of debug, info, warn, error")
	return fallback
}

func (r *Reader) prefixes(key string) []netip.Prefix {
	parts := r.List(key)
	if parts == nil {
		return nil
	}

	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		prefix, err := netip.ParsePrefix(part)
		if err != nil {
			r.Fail(key, "must be comma-separated CIDR prefixes such as 10.0.0.0/8")
			return nil
		}
		prefixes = append(prefixes, prefix)
	}
	return prefixes
}
