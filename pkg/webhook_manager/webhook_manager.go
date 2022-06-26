package webhook_manager

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"go.uber.org/zap"

	"github.com/jirwin/quadlek/pkg/plugin_manager"
	"github.com/jirwin/quadlek/pkg/slack_manager/client"
)

type Config struct {
	ListenAddress string
}

func NewConfig() (Config, error) {
	c := Config{}

	listenAddr := os.Getenv("QUADLEK_WEBHOOK_LISTEN_ADDR")
	if listenAddr == "" {
		return Config{}, fmt.Errorf("QUADLEK_WEBHOOK_LISTEN_ADDR must be set e.g. 0.0.0.0:8000")
	}
	c.ListenAddress = listenAddr

	return c, nil
}

type Manager interface {
	Run(ctx context.Context)
	RegisterRoute(route string, f http.HandlerFunc, methods []string, validateSlack bool)
}

type ManagerImpl struct {
	l             *zap.Logger
	c             Config
	slackConfig   client.Config
	pluginManager plugin_manager.Manager
	server        *http.Server

	router *mux.Router
}

func (m *ManagerImpl) Run(ctx context.Context) {
	m.server.Handler = m.router

	go func() {
		if err := m.server.ListenAndServe(); err != nil {
			m.l.Error("listen error", zap.Error(err))
		}
	}()

	<-ctx.Done()

	m.l.Info("Shutting down webhook server")
	// shut down gracefully, but wait no longer than 5 seconds before halting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = m.server.Shutdown(ctx)
	m.l.Info("Shut down webhook server")
}

func (m *ManagerImpl) RegisterRoute(path string, f http.HandlerFunc, methods []string, validateSlack bool) {
	handler := f
	if validateSlack {
		handler = m.ValidateSlackWebhook(f)
	}

	m.router.HandleFunc(path, handler).Methods(methods...)
	m.l.Info("registering route", zap.String("path", path), zap.Strings("methods", methods))
}

var InvalidRequestSignature = errors.New("invalid request signature")

func (m *ManagerImpl) ValidateSlackWebhook(f http.HandlerFunc) http.HandlerFunc {
	handler := func(rw http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			m.l.Error("error reading request body", zap.Error(err))
			rw.WriteHeader(http.StatusUnauthorized)
			rw.Write(nil)
			return
		}

		rBody := ioutil.NopCloser(bytes.NewBuffer(body))
		r.Body = rBody

		ts := r.Header.Get("X-SlackManager-Request-Timestamp")
		signature := r.Header.Get("X-SlackManager-Signature")

		msg := fmt.Sprintf("v0:%s:%s", ts, body)
		key := []byte(m.slackConfig.VerificationToken)
		h := hmac.New(sha256.New, key)
		h.Write([]byte(msg))
		checkSign := "v0=" + hex.EncodeToString(h.Sum(nil))

		if signature == checkSign {
			f(rw, r)
			return
		}

		rw.WriteHeader(http.StatusUnauthorized)
		rw.Write(nil)
		return
	}

	return handler
}

func New(c Config, l *zap.Logger, slackConfig client.Config, pluginManager plugin_manager.Manager) (*ManagerImpl, error) {
	router := mux.NewRouter()
	m := &ManagerImpl{
		l:             l.Named("webhook-manager"),
		c:             c,
		slackConfig:   slackConfig,
		pluginManager: pluginManager,
		router:        router,
		server: &http.Server{
			Addr:    c.ListenAddress,
			Handler: router,
		},
	}

	return m, nil
}
