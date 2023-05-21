package godev

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func (u ui) devBrowserSTART(url string, wg *sync.WaitGroup) {
	fmt.Println("*** START DEV BROWSER ***")

	ctx, _ := u.CreateContext()
	// defer cancel()

	// crea un mapa para registrar los mensajes de log únicos
	uniqueLogs := make(map[string]bool)

	// captura los logs de JavaScript
	chromedp.ListenTarget(ctx, func(ev interface{}) {
		switch ev := ev.(type) {
		case *runtime.EventConsoleAPICalled:
			for _, arg := range ev.Args {
				s, err := strconv.Unquote(string(arg.Value))
				if err != nil {
					log.Println(err)
					continue
				}
				// verifica si el mensaje de log ya se ha registrado
				if !uniqueLogs[s] {
					uniqueLogs[s] = true
					fmt.Printf("LOG: %s\n", s)
				}
			}
		}
	})

	// Navega a una página web
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		log.Fatal(err)
	}

	// Espera hasta que la página esté completamente cargada
	var loaded bool
	err = chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
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

	go u.ReloadListener(ctx, wg)

	// Cree un canal para recibir señales de interrupción
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	// Espera hasta que se reciba una señal de interrupción o se establezca running en false
	for {
		<-interrupt
		// fmt.Println("Aplicación finalizada")
		// Detenga el navegador y cierre la aplicación cuando se recibe una señal de interrupción
		if err := chromedp.Cancel(ctx); err != nil {
			log.Println("error al cerrar browser", err)
		}
		os.Exit(0)
	}
}

func (u ui) ReloadListener(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		<-u.reload
		// fmt.Println("Recargando Navegador")
		err := chromedp.Run(ctx, chromedp.Reload())
		if err != nil {
			log.Println("Error al recargar Pagina ", err)
		}
	}

}
