//go:generate protoc --go_out=. comics.proto

package comics

import (
  log "github.com/Sirupsen/logrus"
  "github.com/jirwin/quadlek/quadlek"
  "context"
)

func comicCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
  for {
    select {
    case cmdMsg := <-cmdChannel:
      cmdMsg.Command.Reply() <- nil

      case <-ctx.Done():
        log.Info("Exiting comic command.")
      return
    }
  }
}

func Register() quadlek.Plugin {
  return quadlek.MakePlugin(
    "comics",
    []quadlek.Command{
      quadlek.MakeCommand("comic", comicCommand),
    },
    nil,
    nil,
    nil,
    nil,
  )
}
