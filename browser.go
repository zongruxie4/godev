package godev

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chromedp/chromedp"
)

type Browser struct {
	Width    int    // ej "800" default "1024"
	Height   int    //ej: "600" default "768"
	Position string //ej: "1930,0" (when you have second monitor) default: "0,0"

	isOpen bool // Indica si el navegador está abierto

	context.Context    // Este campo no se codificará en YAML
	context.CancelFunc // Este campo no se codificará en YAML

	readyChan chan bool
	errChan   chan error
}

func (h *handler) NewBrowser() {

	h.browser = &Browser{
		readyChan: make(chan bool),
		errChan:   make(chan error),
	}

}

func (h *handler) BrowserPositionAndSizeChanged(fieldName string, oldValue, newValue string) error {

	if !h.browser.isOpen {
		return nil
	}

	err := h.browser.setBrowserPositionAndSize(newValue)
	if err != nil {
		return err
	}

	return h.RestartBrowser()
}

func (h *handler) BrowserStartUrlChanged(fieldName string, oldValue, newValue string) error {

	if !h.browser.isOpen {
		return nil
	}

	return h.RestartBrowser()
}

func (h *handler) RestartBrowser() error {

	this := errors.New("RestartBrowser")

	err := h.CloseBrowser()
	if err != nil {
		return errors.Join(this, err)
	}

	return h.OpenBrowser()
}

func (h *handler) OpenBrowser() error {
	if h.browser.isOpen {
		return errors.New("Browser is already open")
	}

	// Add listener for exit signal
	go func() {
		<-h.exitChan
		h.CloseBrowser()
	}()
	// fmt.Println("*** START DEV BROWSER ***")
	go func() {
		err := h.CreateBrowserContext()
		if err != nil {
			h.browser.errChan <- err
			return
		}

		h.browser.isOpen = true
		var protocol = "http"

		url := protocol + `://localhost:` + h.ch.config.ServerPort + h.ch.config.BrowserStartUrl

		err = chromedp.Run(h.browser.Context, h.browser.sendkeys(url))
		if err != nil {
			h.browser.errChan <- fmt.Errorf("error navigating to %s: %v", h.ch.config.BrowserStartUrl, err)
			return
		}

		// Verificar carga completa
		err = chromedp.Run(h.browser.Context, chromedp.ActionFunc(func(ctx context.Context) error {
			for {
				var readyState string
				select {

				case <-ctx.Done():
					return ctx.Err()
				default:
					err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`document.readyState`, &readyState))
					if err != nil {
						return err
					}

					if readyState == "complete" {
						h.browser.readyChan <- true
						return nil
					}
				}
			}
		}))

		if err != nil {
			h.browser.errChan <- err
		}
	}()

	// Esperar señal de inicio o error
	select {
	case err := <-h.browser.errChan:
		return err
	case <-h.browser.readyChan:
		// Tomar el foco de la TUI después de abrir el navegador
		return h.tui.ReturnFocus()
	}
}

func (h *handler) CreateBrowserContext() error {

	err := h.browser.setBrowserPositionAndSize(h.ch.config.BrowserPositionAndSize)
	if err != nil {
		return err
	}
	// fmt.Printf("tamaño monitor: [%d] x [%d] BrowserPositionAndSize: [%v]\n", width, Height, BrowserPositionAndSize)

	opts := append(

		// select all the elements after the third element
		chromedp.DefaultExecAllocatorOptions[:],
		// chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false), // Desactivar el modo headless

		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,

		//quitar mensaje: Chrome is being controlled by automated test software

		// chromedp.Flag("--webview-log-js-console-messages", true),
		chromedp.WindowSize(h.browser.Width, h.browser.Height),
		chromedp.Flag("window-position", h.browser.Position),
		// chromedp.WindowSize(1530, 870),
		// chromedp.Flag("window-position", "1540,0"),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
		chromedp.Flag("no-focus-on-load", true),
		// chromedp.Flag("exclude-switches", "enable-automation"),
		// chromedp.Flag("disable-blink-features", "AutomationControlled"),
		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,
		// chromedp.Flag("disable-infobars", true),
		// chromedp.Flag("enable-automation", true),
		// chromedp.Flag("disable-infobars", true),
		// chromedp.Flag("exclude-switches", "disable-infobars"),

		chromedp.Flag("disable-blink-features", "WebFontsInterventionV2"), //remove warning font in console [Intervention] Slow network is detected.
		chromedp.Flag("auto-open-devtools-for-tabs", true),
	)

	parentCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	h.browser.Context, h.browser.CancelFunc = chromedp.NewContext(parentCtx)

	return nil
}

func (h *handler) CloseBrowser() error {
	if !h.browser.isOpen {
		return errors.New("Browser is already closed")
	}

	// Primero cerrar todas las pestañas/contextos
	if err := chromedp.Run(h.browser.Context, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			return nil
		}),
	}); err != nil {
		return err
	}

	// Luego cancelar el contexto principal
	if h.browser.CancelFunc != nil {
		h.browser.CancelFunc()
		h.browser.isOpen = false
	}

	// Limpiar recursos
	h.browser.Context = nil
	h.browser.CancelFunc = nil

	return nil
}

func (b Browser) sendkeys(host string) chromedp.Tasks {

	return chromedp.Tasks{
		chromedp.Navigate(host),
	}
}

func (b *Browser) Reload() (err error) {
	if b.Context != nil && b.isOpen {
		// fmt.Println("Recargando Navegador")
		err = chromedp.Run(b.Context, chromedp.Reload())
		if err != nil {
			return errors.New("Reload Browser " + err.Error())
		}
	}
	return
}

func (b *Browser) setBrowserPositionAndSize(newConfig string) (err error) {

	this := errors.New("setBrowserPositionAndSize")

	position, width, height, err := getBrowserPositionAndSize(newConfig)

	if err != nil {
		return errors.Join(this, err)
	}
	b.Position = position

	b.Width, err = strconv.Atoi(width)
	if err != nil {
		return errors.Join(this, err)
	}
	b.Height, err = strconv.Atoi(height)
	if err != nil {
		return errors.Join(this, err)
	}

	return
}

func getBrowserPositionAndSize(config string) (position, width, height string, err error) {
	current := strings.Split(config, ":")

	if len(current) != 2 {
		err = errors.New("Browse Config must be in the format: 1930,0:800,600")
		return
	}

	positions := strings.Split(current[0], ",")
	if len(positions) != 2 {
		err = errors.New("position must be with commas e.g.: 1930,0:800,600")
		return
	}
	position = current[0]

	sizes := strings.Split(current[1], ",")
	if len(sizes) != 2 {
		err = errors.New("width and height must be with commas e.g.: 1930,0:800,600")
		return
	}

	widthInt, err := strconv.Atoi(sizes[0])
	if err != nil {
		err = errors.New("width must be an integer number")
		return
	}
	width = strconv.Itoa(widthInt)

	heightInt, err := strconv.Atoi(sizes[1])
	if err != nil {
		err = errors.New("height must be an integer number")
		return
	}
	height = strconv.Itoa(heightInt)

	return
}

func verifyBrowserPosition(newConfig string) (err error) {
	_, _, _, err = getBrowserPositionAndSize(newConfig)
	return
}
