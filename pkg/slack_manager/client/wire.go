package client

import (
	"github.com/google/wire"
)

var Wired = wire.NewSet(
	NewConfig,
	wire.Struct(new(slackHttpClient)),
	NewSlackClient,
)
