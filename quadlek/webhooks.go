package quadlek

import (
	"context"
	"io"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

func (b *Bot) WebhookServer() {
	mux := http.NewServeMux()

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		io.WriteString(w, "Finished!")
	}))

	srv := &http.Server{Addr: ":8000", Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen: %s\n", err)
		}
	}()

	<-b.ctx.Done()

	log.Info("Shutting down webhook server")
	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	log.Info("Shut down webhook server")
}
