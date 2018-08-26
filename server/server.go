package server

import (
	"context"
	"fmt"
	"github.com/xxlixin1993/LiLGo/configure"
	"github.com/xxlixin1993/LiLGo/graceful"
	"github.com/xxlixin1993/LiLGo/logging"
	"net/http"
	"sync"
	"time"
)

const KHttpServerModuleName = "httpServerModule"

var (
	httpServer *HTTPServer

	NotFoundHandler = func(c Context) error {
		return NewHTTPError(http.StatusNotFound)
	}

	MethodNotAllowedHandler = func(c Context) error {
		return NewHTTPError(http.StatusMethodNotAllowed)
	}
)

type (
	HTTPServer struct {
		host       string
		port       string
		socketLink string
		server     *http.Server
		context    *httpContext
	}

	EasyHandler struct {
		debug  bool
		pool   sync.Pool
		router *Router
		// Handler HTTP error
		HTTPErrorHandler func(error, Context)
	}

	HTTPError struct {
		Code    int
		Message interface{}

		// Extended description
		ExtDes error
	}

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(Context) error
)

// Implement ExitInterface
func (h *HTTPServer) GetModuleName() string {
	return KHttpServerModuleName
}

// Implement ExitInterface
func (h *HTTPServer) Stop() error {
	quitTimeout := configure.DefaultInt("http.quit_timeout", 30)
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(quitTimeout)*time.Second)

	return httpServer.server.Shutdown(ctx)
}

// Run http server
func (eh *EasyHandler) StartHTTPServer() error {
	host := configure.DefaultString("host", "0.0.0.0")
	port := configure.DefaultString("port", "80")
	readTimeout := configure.DefaultInt("http.read_timeout", 4)
	writeTimeout := configure.DefaultInt("http.write_timeout", 3)
	socketLink := host + ":" + port

	httpServer = &HTTPServer{
		host:       host,
		port:       port,
		socketLink: socketLink,
		server: &http.Server{
			Addr:         socketLink,
			Handler:      eh,
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
		},
	}

	// graceful exit
	graceful.GetExitList().Pop(httpServer)

	serveErr := httpServer.server.ListenAndServe()
	if serveErr != nil {
		return serveErr
	}

	return nil
}

// Implements Handler
func (eh *EasyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := eh.pool.Get().(*httpContext)
	ctx.Reset(r, w)

	// TODO middleware

	h := NotFoundHandler
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	eh.router.Find(r.Method, getPath(r), ctx)
	h = ctx.Handler()

	if err := h(ctx); err != nil {
		eh.HTTPErrorHandler(err, ctx)
	}

	eh.pool.Put(ctx)
}

// HTTP error default handler.
// It sends a JSON response with status code.
func (eh *EasyHandler) DefaultHTTPErrorHandler(err error, c Context) {
	var (
		code = http.StatusInternalServerError
		msg  interface{}
	)

	if he, ok := err.(*HTTPError); ok {
		// Http protocol error
		code = he.Code
		msg = he.Message
		if he.ExtDes != nil {
			msg = fmt.Sprintf("%v, %v", err, he.ExtDes)
		}
	} else if eh.debug {
		// Business error
		msg = err.Error()
	} else {
		msg = http.StatusText(code)
	}

	if _, ok := msg.(string); ok {
		data := make(map[string]interface{})
		data["message"] = msg
		msg = data
	}

	// Send response
	if !c.Response().Committed {
		if c.Request().Method == HEAD { // Issue #608
			err = c.NoContent(code)
		} else {
			err = c.JSON(code, msg)
		}
		if err != nil {
			logging.Error(err)
		}
	}
}

// Returns a instance of *httpContext
func (eh *EasyHandler) NewHttpContext(r *http.Request, w http.ResponseWriter) *httpContext {
	return &httpContext{
		request:  r,
		response: NewResponse(w),
	}
}

// TODO middleware
func (eh *EasyHandler) GET(path string, h HandlerFunc) {
	eh.router.Add(GET, path, h)
}

// Returns a instance of *EasyHandler
func NewEasyHandler() *EasyHandler {
	eh := &EasyHandler{
		router: NewRouter(),
		debug:  configure.DefaultBool("app.debug", true),
	}
	eh.pool.New = func() interface{} {
		return eh.NewHttpContext(nil, nil)
	}
	return eh
}

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, message ...interface{}) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(message) > 0 {
		he.Message = message[0]
	}
	return he
}

func getPath(r *http.Request) string {
	path := r.URL.RawPath
	if path == "" {
		path = r.URL.Path
	}
	return path
}

// Error makes it compatible with `error` interface.
func (he *HTTPError) Error() string {
	return fmt.Sprintf("code=%d, message=%v", he.Code, he.Message)
}
