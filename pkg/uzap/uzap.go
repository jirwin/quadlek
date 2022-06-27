package uzap

import (
	"go.uber.org/zap"
	"os"
)

type Config struct {
	Dev bool
}

func NewConfig() (Config, error) {
	c := Config{}

	if os.Getenv("DEV_MODE") != "" {
		c.Dev = true
	}

	return c, nil
}

func New(c Config) (*zap.Logger, error) {
	var logger *zap.Logger
	var err error

	if c.Dev {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	}

	return logger, nil
}
