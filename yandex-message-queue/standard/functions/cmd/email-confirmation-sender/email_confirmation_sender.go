package main

import (
	"encoding/json"
	"fns/reg/internal/service"
	"fns/reg/pkg/httpmock"
	"log"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// func main() {
//
// }

type HandlerRequestBody struct {
	Email string `json:"email"`
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
		Body: &httpmock.ReadCloser{Reader: strings.NewReader(`{"email": "bratushkadan@gmail.com"}`)},
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
	l                       *zap.Logger
	emailConfirmationSender service.EmailConfirmer
}

func (s *httpHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var errs []string

	var b HandlerRequestBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		errs = append(errs, "bad request body, 'email' field required.")
		if err := json.NewEncoder(w).Encode(&HandlerResponseFailure{Errors: errs}); err != nil {
			s.l.Error("failed to serialize error response", zap.Error(err))
		}
		return
	}

	ctx := r.Context()
	if err := s.emailConfirmationSender.Send(ctx, b.Email); err != nil {
		s.l.Error("failed to send confirmation email", zap.Error(err), zap.String("email", b.Email))
		w.WriteHeader(http.StatusInternalServerError)
		errs = append(errs, "failed to send confirmation email")
		if err := json.NewEncoder(w).Encode(&HandlerResponseFailure{Errors: errs}); err != nil {
			s.l.Error("failed to serialize error response", zap.Error(err))
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
		service.NewEmailConfirmationAppConf().LoadEnv(),
		logger,
	)
	if err != nil {
		logger.Fatal(err.Error())
	}

	return &httpHandlerService{
		l:                       logger,
		emailConfirmationSender: svc,
	}
}
