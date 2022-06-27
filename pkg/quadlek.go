package quadlek

import (
	"context"
)

type Quadlek interface {
	Start(ctx context.Context) error
	Stop()
	RegisterPlugin(plugin interface{}) error
}
