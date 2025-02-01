package httpmock

import (
	"io"
	"net/http"
	"os"
)

// MockResponseWriter is a mock for http.ResponseWriter
type MockResponseWriter struct {
	HeaderMap http.Header
	Body      *os.File
	Status    int
}

func NewMockResponseWriter() *MockResponseWriter {
	return &MockResponseWriter{
		HeaderMap: make(http.Header),
		Body:      os.Stdout,
		Status:    http.StatusOK,
	}
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
