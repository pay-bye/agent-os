package config

import (
	"errors"
)

func validate(config Values, input Input) error {
	if input.RequireDatabase && config.DatabaseURL == "" {
		return errors.New("missing_database_url")
	}
	if input.RequireListen && config.Listen == "" {
		return errors.New("missing_listen")
	}
	return nil
}
