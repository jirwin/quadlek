package archive

import (
	"context"
	"os"

	"fmt"

	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/jirwin/ipfs-archive/client/ipfs"
	"github.com/jirwin/ipfs-archive/models"
	"github.com/jirwin/quadlek/quadlek"
)

func archiveCommand(ctx context.Context, cmdChannel <-chan *quadlek.CommandMsg) {
	for {
		select {
		case cmdMsg := <-cmdChannel:
			text := cmdMsg.Command.Text

			// We won't be able to respond immediately, end webhook.
			cmdMsg.Command.Reply() <- nil

			transport := httptransport.New(os.Getenv("IPFS_ARCHIVE_HOST"), "/api", []string{"https"})
			ianClient := ipfs.New(transport, strfmt.Default)
			ianClient.SetTransport(transport)

			resp, err := ianClient.ArchiveURL(ipfs.NewArchiveURLParams().WithContext(ctx).WithBody(&models.ArchiveRequest{
				URL: &text,
			}))
			if err != nil {
				cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
					Text: fmt.Sprintf("Archive request failed: %s", err.Error()),
				})
				continue
			}

			cmdMsg.Bot.RespondToSlashCommand(cmdMsg.Command.ResponseUrl, &quadlek.CommandResp{
				Text: fmt.Sprintf("%s is archived at %s", text, resp.Payload.ArchivedURL),
			})

		case <-ctx.Done():
			return
		}
	}
}

func Register() quadlek.Plugin {
	return quadlek.MakePlugin(
		"ipfs-archive",
		[]quadlek.Command{
			quadlek.MakeCommand("ipfs-archive", archiveCommand),
		},
		nil,
		nil,
		nil,
		nil,
	)
}
