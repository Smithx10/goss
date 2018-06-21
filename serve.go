package goss

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/aelsabbahy/goss/outputs"
	"github.com/aelsabbahy/goss/system"
	"github.com/aelsabbahy/goss/util"
	"github.com/fatih/color"
	"github.com/patrickmn/go-cache"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/urfave/cli"
)

func Serve(c *cli.Context) {
	cache := cache.New(c.Duration("cache"), 30*time.Second)

	health := healthHandler{
		c:             c,
		gossConfig:    getGossConfig(c),
		sys:           system.New(c),
		outputer:      getOutputer(c),
		cache:         cache,
		gossMu:        &sync.Mutex{},
		maxConcurrent: c.Int("max-concurrent"),
	}

	color.NoColor = true

	health.getExitCode()

	if c.String("format") == "prometheus" {
		runHTTPHandler(c, promhttp.Handler())

	} else {

		if c.String("format") == "json" {
			health.contentType = "application/json"
			runHTTPHandler(c, health)
		}
	}
}

func runHTTPHandler(c *cli.Context, handler http.Handler) {

	listenAddr := c.String("listen-addr")
	endpoint := c.String("endpoint")

	http.Handle(endpoint, handler)
	log.Printf("Starting to listen on: %s", listenAddr)
	log.Fatal(http.ListenAndServe(c.String("listen-addr"), nil))

}

type res struct {
	exitCode int
	b        bytes.Buffer
}

type healthHandler struct {
	c             *cli.Context
	gossConfig    GossConfig
	sys           *system.System
	outputer      outputs.Outputer
	cache         *cache.Cache
	gossMu        *sync.Mutex
	contentType   string
	maxConcurrent int
}

func (h healthHandler) getExitCode() (exitCode int, byteBuffer bytes.Buffer) {
	outputConfig := util.OutputConfig{
		FormatOptions: h.c.StringSlice("format-options"),
	}
	iStartTime := time.Now()
	out := validate(h.sys, h.gossConfig, h.maxConcurrent)
	var b bytes.Buffer
	eCode := h.outputer.Output(&b, out, iStartTime, outputConfig)
	return eCode, b
}

func (h healthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Printf("%v: requesting health probe", r.RemoteAddr)
	var resp res
	tmp, found := h.cache.Get("res")
	if found {
		resp = tmp.(res)
	} else {
		h.gossMu.Lock()
		defer h.gossMu.Unlock()
		tmp, found := h.cache.Get("res")
		if found {
			resp = tmp.(res)
		} else {
			h.sys = system.New(h.c)
			log.Printf("%v: Stale cache, running tests", r.RemoteAddr)
			exitCode, b := h.getExitCode()
			resp = res{exitCode: exitCode, b: b}
			h.cache.Set("res", resp, cache.DefaultExpiration)
		}
	}
	if h.contentType != "" {
		w.Header().Set("Content-Type", h.contentType)
	}
	if resp.exitCode == 0 {
		resp.b.WriteTo(w)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		resp.b.WriteTo(w)
	}
}
