package main

import (
	"context"
	"fns/reg/internal/adapters/rest"
	"fns/reg/internal/service"
	"fns/reg/pkg/httpmock"
	"log"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

var (
	emailConfirmationSvc *rest.Adapter
)

func main() {
	w := httpmock.NewMockResponseWriter()
	r := &http.Request{
		Body: &httpmock.ReadCloser{Reader: strings.NewReader(`{"token": "ouy4ukricebc2v4og46smanv5e7g5ty6lsyqfp24bkhpv74wgnabvpfd5lwyzf2nca5vyl2gjy3voyetj2bmffoz6j7c5wakslcguoa"}`)},
	}
	Handler(w, r)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	emailConfirmationSvc.HandleConfirmEmail(w, r)
}

func init() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	conf := service.NewConf().
		WithDocYdb().
		WithSqs().
		Build()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	svc, err := service.New(
		ctx,
		conf,
		service.WithLogger(logger),
		service.WithDynamoDb(),
		service.WithYmq(),
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	emailConfirmationSvc = rest.New(svc, logger)
}
