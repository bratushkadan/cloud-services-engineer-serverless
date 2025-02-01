package main

import (
	"encoding/json"
	"errors"
	"fns/reg/internal/service"
	"fns/reg/pkg/httpmock"
	"log"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type HandlerRequestBody struct {
	Token string `json:"token"`
}

type HandlerResponseSuccess struct {
	Ok bool `json:"ok"`
}
type HandlerResponseFailure struct {
	Errors []string `json:"errors"`
}

func main() {
	w := httpmock.NewMockResponseWriter()
	r := &http.Request{
		Body: &httpmock.ReadCloser{Reader: strings.NewReader(`{"token": "x6uaruup4xqiyjaemg7nzwwe22gfuppnytv6hgxvhyeak3eofvzt5bk7zackkodn7witwipceut4kq2b5p3voi5rxm2p7zg3uxolawy"}`)},
	}
	Handler(w, r)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	executer := mustPrepareExecuter()
	executer.ServeHTTP(w, r)
}

type Executer interface {
	http.Handler
}

type httpHandlerService struct {
	l              *zap.Logger
	emailConfirmer service.EmailConfirmer
}

func (s *httpHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var errs []string

	var b HandlerRequestBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		errs = append(errs, "bad request body, 'token' field required.")
		w.WriteHeader(http.StatusBadRequest)
		if err := json.NewEncoder(w).Encode(&HandlerResponseFailure{Errors: errs}); err != nil {
			s.l.Error("failed to serialize response", zap.Error(err))
		}
		return
	}

	ctx := r.Context()
	if err := s.emailConfirmer.Confirm(ctx, b.Token); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidConfirmationToken) || errors.Is(err, service.ErrConfirmationTokenExpired):
			w.WriteHeader(http.StatusBadRequest)
			errs = append(errs, err.Error())
		default:
			s.l.Error("failed to confirm email", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			errs = append(errs, "failed to confirm email")
		}

		if err := json.NewEncoder(w).Encode(&HandlerResponseFailure{Errors: errs}); err != nil {
			s.l.Error("failed to serialize response", zap.Error(err))
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}

func mustPrepareExecuter() Executer {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal(err)
	}

	svc, err := service.NewEmailConfirmation(
		service.NewEmailConfirmationAppConf().WithSqs().LoadEnv(),
		logger,
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return &httpHandlerService{
		l:              logger,
		emailConfirmer: svc,
	}
}
