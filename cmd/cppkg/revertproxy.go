package main

import (
	"bytes"
	stdlog "log"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

type revertProxy = httptest.Server

func startRevertProxy(endpoint string, log *stdlog.Logger) (_ *revertProxy, err error) {
	rpURL, err := url.Parse(endpoint)
	if err != nil {
		return
	}
	if log == nil {
		log = stdlog.Default()
	}
	proxy := httptest.NewServer(&httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			r.SetURL(rpURL)
			var b bytes.Buffer
			r.Out.Write(&b)
			log.Print(b.String())
		},
	})
	return proxy, nil
}

const (
	conanCenter   = "conancenter"
	conanEndpoint = "https://center2.conan.io"
)

func (p *Manager) remoteProxy(flags int, logfile string, f func() error) (err error) {
	const conanCenterOld = conanCenter + ".origin"

	quietInstall := flags&ToolQuietInstall != 0
	app, err := conanCmd.Get(quietInstall)
	if err != nil {
		return
	}

	cmd := exec.Command(app, "remote", "rename", conanCenter, conanCenterOld)
	result, err := cmd.CombinedOutput()
	if err != nil {
		if !strings.Contains(string(result), "already exists") {
			return
		}
	}
	defer func() {
		exec.Command(app, "remote", "remove", conanCenter).Run()
		exec.Command(app, "remote", "rename", conanCenterOld, conanCenter).Run()
	}()

	var log *stdlog.Logger
	if logfile != "" {
		f, err := os.Create(logfile)
		if err == nil {
			defer f.Close()
			log = stdlog.New(f, "", stdlog.LstdFlags)
		}
	}
	rp, err := startRevertProxy(conanEndpoint, log)
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
