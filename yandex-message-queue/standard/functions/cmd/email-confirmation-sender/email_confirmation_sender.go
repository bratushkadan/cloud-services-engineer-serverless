package main

import (
	"bytes"
	"encoding/json"
	"fns/reg/internal/service"
	"io"
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

type HandlerResponse struct {
	Ok bool `json:"ok"`
}

// TODO: move to pkg testing

// MockResponseWriter is a mock for http.ResponseWriter
type MockResponseWriter struct {
	HeaderMap http.Header
	Body      *bytes.Buffer
	Status    int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
		Status:    http.StatusOK,
	}
}

func (m *MockResponseWriter) BodyBuffer() *bytes.Buffer {
	return m.Body
}

func (m *MockResponseWriter) Header() http.Header {
	return m.HeaderMap
}

func (m *MockResponseWriter) Write(b []byte) (int, error) {
	return m.Body.Write(b)
}

func (m *MockResponseWriter) WriteHeader(statusCode int) {
	m.Status = statusCode
}

// ReadCloser wraps a strings.Reader to provide a no-op Close method.
type ReadCloser struct {
	io.Reader
}

// Close implements the io.Closer interface by adding a no-op Close method.
func (ReadCloser) Close() error {
	return nil
}

// ENDTODO

func main() {
	w := NewMockResponseWriter()
	r := &http.Request{
		Body: &ReadCloser{Reader: strings.NewReader(`{"email": "bratushkadan@gmail.com"}`)},
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
	emailConfirmationSender service.EmailConfirmationSender
}

func (s *httpHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var b HandlerRequestBody
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":["bad request body, 'email' field required."]}`))
		return
	}

	ctx := r.Context()
	if err := s.emailConfirmationSender.Send(ctx, b.Email); err != nil {
		s.l.Error("failed to send confirmation email", zap.Error(err), zap.String("email", b.Email))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors": ["failed to send confirmation email"]}`))
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
