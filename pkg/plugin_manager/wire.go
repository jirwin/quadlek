package plugin_manager

import "github.com/google/wire"

var Wired = wire.NewSet(
	NewConfig,
	wire.Struct(new(ManagerImpl), "*"),
)
