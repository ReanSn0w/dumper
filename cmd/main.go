package main

import (
	"bytes"
	"fmt"
	"io"
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
		Disabled bool   `long:"enabled" env:"ENABLED" description:"Disable logging mode (service will work as a proxy)"`
		Target   string `long:"target" env:"TARGET" description:"target host"`
		Body     bool   `long:"print-body" env:"PRINT_BODY" description:"print body"`
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

func handler(log lgr.L, pass *url.URL) http.Handler {
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

		r.URL.Host = pass.Host
		r.URL.Scheme = pass.Scheme
		r.Host = pass.Host
		r.RequestURI = ""

		dumpRequest, dumpRequestErr = httputil.DumpRequest(r, opts.Body)

		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			web.NewResponse(err).Write(http.StatusInternalServerError, w)
			return
		}

		dumpResponse, dumpResponseErr = httputil.DumpResponse(resp, opts.Body)

		w.WriteHeader(resp.StatusCode)
		for key, vals := range resp.Header {
			for _, val := range vals {
				w.Header().Add(key, val)
			}
		}

		io.Copy(w, resp.Body)
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

	if len(resp) != 0 || respErr != nil {
		buffer.WriteString("\n\nResponse:\n\n")
		if respErr == nil {
			buffer.Write(resp)
		} else {
			buffer.WriteString(fmt.Sprintf("Error: %v", respErr))
		}
	}

	buffer.WriteString("\n\n")
	return buffer.String()
}
