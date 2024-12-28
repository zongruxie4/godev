package godev

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/chromedp/chromedp"
	"github.com/fstanis/screenresolution"
)

type Browser struct {
	HomePath string // ej: "/index.html", "/login", default: "/"
	Port     int    // ej: 4430 default 8080,
	With     int    // ej "800" default "1024"
	Height   int    //ej: "600" default "768"
	Position string //ej: "1930,0" (when you have second monitor) default: "0,0"

	context.Context    // Este campo no se codificará en YAML
	context.CancelFunc // Este campo no se codificará en YAML

}

func (b *Browser) SetDefault() error {

	b.HomePath = "/"
	b.Port = 8080
	b.Position = "0,0"

	err := b.SetScreenSize()

	return err
}

func (b Browser) ConfigTemplateContent() string {

	return `# browser config:
homePath: ` + b.HomePath + `
port: ` + strconv.Itoa(b.Port) + `
with: ` + strconv.Itoa(b.With) + `
height: ` + strconv.Itoa(b.Height) + `
position: ` + b.Position

}

func New() *Browser {

	b := Browser{}

	return &b
}

func (b *Browser) BrowserStart(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("*** START DEV BROWSER ***")

	b.CreateBrowserContext()
	// defer cancel()
	var protocol = "http"

	// Convertir el puerto a una cadena de texto
	portStr := strconv.Itoa(b.Port)

	if strings.Contains(portStr, "44") {
		protocol = "https"
	}

	// Navega a una página web

	url := protocol + `://localhost:` + portStr + b.HomePath

	err := chromedp.Run(b.Context, b.sendkeys(url)) // chromedp.Navigate(url),

	if err != nil {
		log.Fatal("Error al navegar "+b.HomePath+" ", err)
	}

	// Espera hasta que la página esté completamente cargada
	var loaded bool

	err = chromedp.Run(b.Context, chromedp.ActionFunc(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				var readyState string
				err := chromedp.Run(ctx, chromedp.EvaluateAsDevTools(`document.readyState`, &readyState))
				if err != nil {
					return err
				}

				if readyState == "complete" {
					loaded = true

					return nil
				}
			}
		}
	}))
	if err != nil {
		log.Fatal(err)
	}

	// Verifica si la página se ha cargado correctamente
	if !loaded {
		log.Fatal("La página no se ha cargado correctamente")
	}

}

func (b *Browser) CreateBrowserContext() {
	var Position = "0,0" // ej: "1930,0"

	if b.Position != "" {
		Position = b.Position
	}

	// fmt.Printf("tamaño monitor: [%d] x [%d] Position: [%v]\n", width, Height, Position)

	opts := append(

		// select all the elements after the third element
		chromedp.DefaultExecAllocatorOptions[:],
		// chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false), // Desactivar el modo headless

		// chromedp.NoFirstRun,
		// chromedp.NoDefaultBrowserCheck,

		//quitar mensaje: Chrome is being controlled by automated test software

		// chromedp.Flag("--webview-log-js-console-messages", true),
		chromedp.WindowSize(b.With, b.Height),
		chromedp.Flag("window-Position", Position),
		// chromedp.WindowSize(1530, 870),
		// chromedp.Flag("window-Position", "1540,0"),
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
}
func (b Browser) BrowserContext() context.Context {
	return b.Context
}

func (b Browser) sendkeys(host string) chromedp.Tasks {

	return chromedp.Tasks{
		chromedp.Navigate(host),
	}
}

func (b *Browser) Reload() (err string) {
	if b.Context != nil {
		// fmt.Println("Recargando Navegador")
		er := chromedp.Run(b.Context, chromedp.Reload())
		if er != nil {
			return "Reload error al recargar Pagina " + er.Error()
		}
	}
	return
}

func (b *Browser) SetScreenSize() error {

	r := screenresolution.GetPrimary()
	if r == nil {
		return errors.New("error SetScreenSize sistema operativo no soportado")
	}

	b.With = r.Width
	b.Height = r.Height

	return nil

}
