package server

import (
	"bufio"
	"github.com/xxlixin1993/LiLGo/logging"
	"net"
	"net/http"
)

// Response implements an http.ResponseWriter interface.
// Its an HTTP handler to construct an HTTP response.
type Response struct {
	beforeFuncs []func()
	afterFuncs  []func()
	Writer      http.ResponseWriter
	Status      int
	Size        int64
	Committed   bool
}

func NewResponse(w http.ResponseWriter) (r *Response) {
	return &Response{Writer: w}
}

// Implements ResponseWriter
func (response *Response) Header() http.Header {
	return response.Writer.Header()
}

// Implements ResponseWriter
func (response *Response) Write(b []byte) (int, error) {
	if !response.Committed {
		response.WriteHeader(http.StatusOK)
	}
	n, err := response.Writer.Write(b)
	response.Size += int64(n)
	for _, fn := range response.afterFuncs {
		fn()
	}
	return n, err
}

// Implements ResponseWriter
func (response *Response) WriteHeader(statusCode int) {
	if response.Committed {
		logging.Warning("response already committed")
		return
	}
	for _, fn := range response.beforeFuncs {
		fn()
	}
	response.Status = statusCode
	response.Writer.WriteHeader(statusCode)
	response.Committed = true
}

// Registers a function which is called just before the response is written.
func (response *Response) Before(fn func()) {
	response.beforeFuncs = append(response.beforeFuncs, fn)
}

// Registers a function which is called just after the response is written.
func (response *Response) After(fn func()) {
	response.afterFuncs = append(response.afterFuncs, fn)
}

// Implements Flusher
func (response *Response) Flush() {
	response.Writer.(http.Flusher).Flush()
}

// Implements the http.Hijacker interface to allow an HTTP handler to
// take over the connection.
func (response *Response) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return response.Writer.(http.Hijacker).Hijack()
}

// Reset Response struct
func (response *Response) reset(w http.ResponseWriter) {
	response.beforeFuncs = nil
	response.afterFuncs = nil
	response.Writer = w
	response.Size = 0
	response.Status = http.StatusOK
	response.Committed = false
}
