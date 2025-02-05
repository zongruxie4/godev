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

func NewBrowser() *Browser {

	b := &Browser{
		readyChan: make(chan bool),
		errChan:   make(chan error),
	}

	config.Subscribe(b)

	return b
}

func (b *Browser) OnConfigChanged(fieldName string, oldValue, newValue string) {

	if !b.isOpen {
		return
	}

	return
}

func (b *Browser) OpenBrowser() error {
	if b.isOpen {
		return errors.New("Browser is already open")
	}

	// Add listener for exit signal
	go func() {
		<-exitChan
		b.CloseBrowser()
	}()
	// fmt.Println("*** START DEV BROWSER ***")
	go func() {
		err := b.CreateBrowserContext()
		if err != nil {
			b.errChan <- err
			return
		}

		b.isOpen = true
		var protocol = "http"

		url := protocol + `://localhost:` + config.ServerPort + config.BrowserStartUrl

		err = chromedp.Run(b.Context, b.sendkeys(url))
		if err != nil {
			b.errChan <- fmt.Errorf("error navigating to %s: %v", config.BrowserStartUrl, err)
			return
		}

		// Verificar carga completa
		err = chromedp.Run(b.Context, chromedp.ActionFunc(func(ctx context.Context) error {
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
						b.readyChan <- true
						return nil
					}
				}
			}
		}))

		if err != nil {
			b.errChan <- err
		}
	}()

	// Esperar señal de inicio o error
	select {
	case err := <-b.errChan:
		return err
	case <-b.readyChan:
		return nil
	}
}

func (b *Browser) CreateBrowserContext() error {

	err := b.setBrowserPositionAndSize(config.BrowserPositionAndSize)
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
		chromedp.WindowSize(b.Width, b.Height),
		chromedp.Flag("window-BrowserPositionAndSize", b.Position),
		// chromedp.WindowSize(1530, 870),
		// chromedp.Flag("window-BrowserPositionAndSize", "1540,0"),
		chromedp.Flag("use-fake-ui-for-media-stream", true),
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

	b.Context, b.CancelFunc = chromedp.NewContext(parentCtx)

	return nil
}

func (b *Browser) CloseBrowser() error {
	if !b.isOpen {
		return errors.New("Browser is already closed")
	}

	// Primero cerrar todas las pestañas/contextos
	if err := chromedp.Run(b.Context, chromedp.Tasks{
		chromedp.ActionFunc(func(ctx context.Context) error {
			return nil
		}),
	}); err != nil {
		return err
	}

	// Luego cancelar el contexto principal
	if b.CancelFunc != nil {
		b.CancelFunc()
		b.isOpen = false
	}

	// Limpiar recursos
	b.Context = nil
	b.CancelFunc = nil

	return nil
}

func (b Browser) sendkeys(host string) chromedp.Tasks {

	return chromedp.Tasks{
		chromedp.Navigate(host),
	}
}

func (b *Browser) Reload() (err error) {
	if b.Context != nil {
		// fmt.Println("Recargando Navegador")
		err = chromedp.Run(b.Context, chromedp.Reload())
		if err != nil {
			return errors.New("Reload error al recargar Pagina " + err.Error())
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
