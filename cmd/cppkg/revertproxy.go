package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	stdlog "log"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

var (
	errPassThrough = errors.New("pass through")
)

type rtFunc = func(req *http.Request) (resp *http.Response, err error)

func rtDefault(req *http.Request) (resp *http.Response, err error) {
	return nil, errPassThrough
}

type rtHandler rtFunc

func (p rtHandler) RoundTrip(req *http.Request) (*http.Response, error) {
	return p(req)
}

type teeReader struct {
	rc   io.ReadCloser
	b    bytes.Buffer
	req  *http.Request
	resp *http.Response
	log  *stdlog.Logger
}

func (p *teeReader) Read(b []byte) (n int, err error) {
	n, err = p.rc.Read(b)
	p.b.Write(b[:n])
	return
}

func (p *teeReader) Close() error {
	err := p.rc.Close()
	resp := *p.resp
	resp.Body = io.NopCloser(&p.b)
	var b bytes.Buffer
	p.req.Write(&b)
	resp.Write(&b)
	p.log.Print(b.String())
	return err
}

type revertProxy = httptest.Server

func startRevertProxy(endpoint string, rt rtFunc, log *stdlog.Logger) (_ *revertProxy, err error) {
	rpURL, err := url.Parse(endpoint)
	if err != nil {
		return
	}
	if log == nil {
		log = stdlog.Default()
	}
	if rt == nil {
		rt = rtDefault
	}
	proxy := httptest.NewServer(&httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(rpURL)
		},
		Transport: rtHandler(func(req *http.Request) (resp *http.Response, err error) {
			resp, err = rt(req)
			if err == errPassThrough {
				resp, err = http.DefaultTransport.RoundTrip(req)
			}
			if resp.Body != nil {
				resp.Body = &teeReader{
					rc:   resp.Body,
					req:  req,
					resp: resp,
					log:  log,
				}
			}
			return
		}),
	})
	return proxy, nil
}

const (
	conanCenter   = "conancenter"
	conanEndpoint = "https://center2.conan.io"
)

type remoteList []struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

func remoteProxy(flags int, logfile string, f func() error, rt rtFunc) (err error) {
	quietInstall := flags&ToolQuietInstall != 0
	app, err := conanCmd.Get(quietInstall)
	if err != nil {
		return
	}

	endpoint := conanEndpoint
	cmd := exec.Command(app, "remote", "list", "-f", "json")
	if b, err := cmd.Output(); err == nil {
		var rl remoteList
		if json.Unmarshal(b, &rl) == nil {
			for _, r := range rl {
				if r.Name == conanCenter && strings.HasPrefix(r.URL, "https://") {
					endpoint = r.URL
					break
				}
			}
		}
	}
	defer func() {
		exec.Command(app, "remote", "add", "--force", conanCenter, endpoint).Run()
	}()

	var log *stdlog.Logger
	if logfile != "" {
		f, err := os.Create(logfile)
		if err == nil {
			defer f.Close()
			log = stdlog.New(f, "", stdlog.LstdFlags)
		}
	}
	rp, err := startRevertProxy(conanEndpoint, rt, log)
	if err != nil {
		return
	}
	defer rp.Close()

	err = exec.Command(app, "remote", "add", "--force", conanCenter, rp.URL).Run()
	if err != nil {
		return
	}

	return f()
}
