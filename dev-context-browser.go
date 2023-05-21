package godev

import (
	"context"

	"github.com/chromedp/chromedp"
)

func (u ui) CreateContext() (context.Context, context.CancelFunc) {

	opts := append(

		// select all the elements after the third element
		chromedp.DefaultExecAllocatorOptions[:],
		// chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false), // Desactivar el modo headless

		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("auto-open-devtools-for-tabs", true),
		//quitar mensaje: Chrome is being controlled by automated test software

		// chromedp.Flag("--webview-log-js-console-messages", true),
		chromedp.WindowSize(1530, 870),
		chromedp.Flag("window-position", "1540,0"),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		// chromedp.Flag("exclude-switches", "enable-automation"),
		// chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,
		// chromedp.Flag("disable-infobars", true),
		// chromedp.Flag("enable-automation", true),
		// chromedp.Flag("disable-infobars", true),
		// chromedp.Flag("exclude-switches", "disable-infobars"),
	)

	parentCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)
	return chromedp.NewContext(parentCtx)
}
