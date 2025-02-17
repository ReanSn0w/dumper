package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/ReanSn0w/gokit/pkg/app"
	"github.com/ReanSn0w/gokit/pkg/web"
	"github.com/go-pkgz/lgr"
)

var (
	revision = "unknown"
	opts     = struct {
		app.Debug

		Port     int    `long:"port" env:"PORT" default:"8080" description:"Port to listen on"`
		Disabled bool   `long:"enabled" env:"ENABLED" description:"Disable logging mode"`
		Target   string `long:"target" env:"TARGET" description:"Target host for logging"`
		Body     bool   `long:"debug-body" env:"DEBUG_BODY" description:"Debug body"`
	}{}
)

func main() {
	app := app.New("Debug Container", revision, &opts)

	{
		proxyURL, err := url.Parse(opts.Target)
		if err != nil {
			panic(err)
		}

		srv := web.New(app.Log())
		srv.Run(app.CancelCause(), opts.Port, handler(app.Log(), proxyURL))
	}

	app.GracefulShutdown(time.Second * 3)
}

func handler(log lgr.L, url *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			dumpRequest     []byte
			dumpRequestErr  error
			dumpResponse    []byte
			dumpResponseErr error
		)

		defer func() {
			if !opts.Disabled {
				print := makePrint(dumpRequest, dumpResponse, dumpRequestErr, dumpResponseErr)
				log.Logf("[INFO] %s", print)
			}
		}()

		dumpRequest, dumpRequestErr = httputil.DumpRequest(r, opts.Body)

		proxy := httputil.ReverseProxy{
			Director: func(req *http.Request) {
				req.URL.Scheme = url.Scheme
				req.URL.Host = url.Host
				req.URL.User = url.User
				req.URL.Path = r.URL.Path
				req.URL.RawQuery = r.URL.RawQuery
			},
			ModifyResponse: func(r *http.Response) error {
				dumpResponse, dumpResponseErr = httputil.DumpResponse(r, opts.Body)
				return nil
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				web.NewResponse(err).Write(http.StatusInternalServerError, w)
			},
		}

		proxy.ServeHTTP(w, r)
	})
}

func makePrint(req, resp []byte, reqErr, respErr error) string {
	buffer := new(bytes.Buffer)

	buffer.WriteString("----------\n")
	buffer.WriteString("Request: \n\n")
	if reqErr == nil {
		buffer.Write(req)
	} else {
		buffer.WriteString(fmt.Sprintf("Error: %v", reqErr))
	}
	buffer.WriteString("\n\nResponse:\n\n")
	if respErr == nil {
		buffer.Write(resp)
	} else {
		buffer.WriteString(fmt.Sprintf("Error: %v", respErr))
	}
	buffer.WriteString("\n\n")

	return buffer.String()
}
