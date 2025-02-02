package main

import (
	"context"
	"fns/reg/internal/adapters/rest"
	"fns/reg/internal/service"
	"log"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var (
	emailConfirmationSvc *rest.Adapter
)

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
