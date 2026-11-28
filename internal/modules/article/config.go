package article

import (
	"github.com/snykk/go-rest-boilerplate/internal/platform/config"
)

// Config holds the article module's own settings, read from ARTICLE_* variables.
type Config struct {
	// DefaultPageSize is the number of articles a list returns when the request asks for none.
	DefaultPageSize int
	// MaxPageSize is the most articles a list returns.
	MaxPageSize int
}

// LoadConfig reads ARTICLE_DEFAULT_PAGE_SIZE (default 20) and ARTICLE_MAX_PAGE_SIZE (default 100) from source. It
// reports every setting that is not a positive integer, and a default page size above the maximum, in one
// error.
func LoadConfig(source config.Source) (Config, error) {
	reader := config.NewReader(source)
	settings := Config{
		DefaultPageSize: int(reader.PositiveInt("ARTICLE_DEFAULT_PAGE_SIZE", 20)),
		MaxPageSize:     int(reader.PositiveInt("ARTICLE_MAX_PAGE_SIZE", 100)),
	}
	if settings.DefaultPageSize > settings.MaxPageSize {
		reader.Fail("ARTICLE_DEFAULT_PAGE_SIZE", "must not exceed ARTICLE_MAX_PAGE_SIZE")
	}

	if err := reader.Err(); err != nil {
		return Config{}, err
	}
	return settings, nil
}
