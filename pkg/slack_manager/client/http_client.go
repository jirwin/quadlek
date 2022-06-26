package client

import (
	"bytes"
	"go.uber.org/zap"
	"io/ioutil"
	"net/http"
)

type slackHttpClient struct {
	L *zap.Logger
	C Config
}

func (httpClient *slackHttpClient) Do(req *http.Request) (*http.Response, error) {
	client := &http.Client{}
	r, e := client.Do(req)
	if r != nil {
		data, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		buffer := bytes.NewBuffer(data)

		r.Body = ioutil.NopCloser(buffer)

		if httpClient.C.RequestTracing {
			httpClient.L.Debug("request complete", zap.String("payload", string(data)))
		}
	}
	return r, e
}
