// Package ariutil provides the connection to Asterisk's ARI (Asterisk REST
// Interface) server, the bridge between FRED and the Asterisk PBX.
package ariutil

import (
	"fmt"
	"os"

	"github.com/CyCoreSystems/ari/v5"
	"github.com/CyCoreSystems/ari/v5/client/native"
)

// Config holds the connection parameters needed to reach an Asterisk ARI server.
type Config struct {
	Application  string
	Username     string
	Password     string
	URL          string
	WebsocketURL string
}

// ConfigFromEnv builds a Config from the standard ARI_* environment variables.
func ConfigFromEnv() Config {
	return Config{
		Application:  os.Getenv("ARI_APPLICATION_NAME"),
		Username:     os.Getenv("ARI_USERNAME"),
		Password:     os.Getenv("ARI_PASSWORD"),
		URL:          os.Getenv("ARI_URL"),
		WebsocketURL: os.Getenv("ARI_WS_URL"),
	}
}

// Validate checks that every required field is present, so a missing
// environment variable fails fast with a clear message.
func (c Config) Validate() error {
	var missing []string
	for name, val := range map[string]string{
		"ARI_APPLICATION_NAME": c.Application,
		"ARI_USERNAME":         c.Username,
		"ARI_PASSWORD":         c.Password,
		"ARI_URL":              c.URL,
		"ARI_WS_URL":           c.WebsocketURL,
	} {
		if val == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ariutil: missing required environment variables: %v", missing)
	}
	return nil
}

// NewARIClient connects to Asterisk's ARI server using the given config.
func NewARIClient(cfg Config) (ari.Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	cl, err := native.Connect(&native.Options{
		Application:  cfg.Application,
		Username:     cfg.Username,
		Password:     cfg.Password,
		URL:          cfg.URL,
		WebsocketURL: cfg.WebsocketURL,
	})
	if err != nil {
		return nil, fmt.Errorf("ariutil: connect to ARI: %w", err)
	}
	return cl, nil
}
