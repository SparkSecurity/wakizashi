package scrape

import (
	"context"
	"github.com/SparkSecurity/wakizashi/worker/config"
	"github.com/SparkSecurity/wakizashi/worker/storage"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"log"
	"os"
	"strings"
	"time"
)

var browserCtx context.Context
var browserCancel context.CancelFunc

func InitBrowserNew() {
	options := append(chromedp.DefaultExecAllocatorOptions[:],
		// block all images
		chromedp.Flag("blink-settings", "imagesEnabled=false"),
		chromedp.Flag("ignore-certificate-errors", "true"),
		chromedp.Flag("headless", "new"),
		chromedp.ProxyServer(os.Getenv("PROXY")),
	)
	allocatorCtx, _ := chromedp.NewExecAllocator(context.Background(), options...)
	browserCtx, browserCancel = chromedp.NewContext(allocatorCtx)
}

func DisableUselessResources(ctx context.Context) func(event interface{}) {
	return func(event interface{}) {
		switch ev := event.(type) {
		case *fetch.EventRequestPaused:
			go func() {
				c := chromedp.FromContext(ctx)
				ctx := cdp.WithExecutor(ctx, c.Target)
				if ev.ResourceType == network.ResourceTypeImage ||
					ev.ResourceType == network.ResourceTypeFont ||
					ev.ResourceType == network.ResourceTypeMedia ||
					ev.ResourceType == network.ResourceTypeStylesheet {
					fetch.FailRequest(ev.RequestID, network.ErrorReasonBlockedByClient).Do(ctx)
				} else {
					fetch.ContinueRequest(ev.RequestID).Do(ctx)
				}
			}()
		}
	}
}

func HandlerBrowserNew(task *Task) error {
	var html string
	pageCtx, cancel := chromedp.NewContext(browserCtx)
	defer cancel()
	chromedp.ListenTarget(pageCtx, DisableUselessResources(pageCtx))

	log.Println("navigate url", task.Url)
	err := chromedp.Run(pageCtx,
		fetch.Enable(),
		chromedp.Navigate(task.Url),
		chromedp.Sleep(time.Duration(config.Config.BrowserWait)*time.Second),
	)
	if err != nil {
		log.Println("cannot navigate url", task.Url, err)
		return err
	}
	waitReadyCtx, cancel := context.WithTimeout(pageCtx, time.Duration(config.Config.BrowserTimeout)*time.Second)
	defer cancel()
	_ = chromedp.Run(waitReadyCtx,
		chromedp.WaitReady("html"),
	)
	err = chromedp.Run(pageCtx,
		chromedp.OuterHTML("html", &html),
	)
	if err != nil {
		return err
	}
	fileId, err := storage.Storage.UploadFile(strings.NewReader(html))
	if err != nil {
		return err
	}
	task.Response = fileId
	return nil
}

func CloseBrowserNew() {
	browserCancel()
}
