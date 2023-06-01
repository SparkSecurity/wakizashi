package main

import (
	"errors"
	"github.com/projectdiscovery/gologger"
	"github.com/projectdiscovery/katana/pkg/engine/hybrid"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"
	"log"
	"net/http"
	"time"
)

var callbackMap = make(map[string]func(result *output.Result, err error))
var Crawler *hybrid.Crawler

func ScrapeInit() {
	options := &types.Options{
		MaxDepth:     1,                // Maximum depth to crawl
		FieldScope:   "rdn",            // Crawling Scope Field
		BodyReadSize: 15 * 1024 * 1024, // Maximum response size to read
		RateLimit:    150,              // Maximum requests to send per second
		Strategy:     "depth-first",    // Visit strategy (depth-first, breadth-first)
		Timeout:      10,
		Proxy:        Config.Proxy,
		OnResult: func(result output.Result) { // Callback function to execute for result
			if callbackMap[result.Request.URL] == nil {
				log.Println("No callback for url", result.Request.URL)
				return
			}
			if result.Error != "" {
				callbackMap[result.Request.URL](nil, errors.New(result.Error))
				delete(callbackMap, result.Request.URL)
				return
			}
			if result.Response.StatusCode >= 400 {
				callbackMap[result.Request.URL](nil, errors.New(http.StatusText(result.Response.StatusCode)))
				delete(callbackMap, result.Request.URL)
				return
			}
			callbackMap[result.Request.URL](&result, nil)
			delete(callbackMap, result.Request.URL)
		},
		Headless: true,
	}
	crawlerOptions, err := types.NewCrawlerOptions(options)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}
	defer crawlerOptions.Close()
	Crawler, err = hybrid.New(crawlerOptions)
	if err != nil {
		gologger.Fatal().Msg(err.Error())
	}
}

func ScrapeHandler(task *ScrapeTask) error {
	ch := make(chan *output.Result)
	errCh := make(chan error)
	oldCallback := callbackMap[task.Url]
	callbackMap[task.Url] = func(result *output.Result, err error) {
		if err != nil {
			errCh <- err
			return
		}
		ch <- result
		if oldCallback != nil {
			oldCallback(result, err)
		}
	}
	go func() {
		err := Crawler.Crawl(task.Url)
		if err != nil {
			errCh <- err
		}
	}()
	select {
	case result := <-ch:
		task.Response = result.Response.Body
	case err := <-errCh:
		return err
	case <-time.After(17 * time.Second):
		return errors.New("timeout")
	}
	return nil
}

func ScrapeClose() {
	_ = Crawler.Close()
}
