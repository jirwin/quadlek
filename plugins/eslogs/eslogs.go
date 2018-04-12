package eslogs

import (
	"context"

	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/jirwin/quadlek/quadlek"
	"gopkg.in/olivere/elastic.v5"
)

var (
	esEndpoint = ""
	esIndex    = ""
	esClient   *elastic.Client
)

func logHook(ctx context.Context, hookchan <-chan *quadlek.HookMsg) {
	for {
		select {
		case hookMsg := <-hookchan:
			_, err := esClient.Index().Index(esIndex).Type("slack-msg").Id(hookMsg.Msg.Timestamp).BodyJson(hookMsg.Msg).Do(ctx)
			if err != nil {
				log.WithError(err).Error("Error indexing log to ES")
				continue
			}

		case <-ctx.Done():
			log.Info("Exiting es log hook")
			return
		}
	}
}

func Register(endpoint, index string) (quadlek.Plugin, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("es endpoint is required")
	}
	esEndpoint = endpoint

	if index == "" {
		return nil, fmt.Errorf("es index is required")
	}
	esIndex = index

	esc, err := elastic.NewClient(elastic.SetURL(esEndpoint), elastic.SetSniff(false))
	if err != nil {
		return nil, err
	}
	esClient = esc

	return quadlek.MakePlugin(
		"eslogs",
		nil,
		[]quadlek.Hook{
			quadlek.MakeHook(logHook),
		},
		nil,
		nil,
		nil,
	), nil
}
