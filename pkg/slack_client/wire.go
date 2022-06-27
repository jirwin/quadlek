package slack_client

import (
	"github.com/google/wire"
)

var Wired = wire.NewSet(
	NewConfig,
	wire.Struct(new(SlackHttpClient)),
	NewSlackClient,
)
