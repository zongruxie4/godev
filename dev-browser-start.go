package godev

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"

	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

func (a *Args) DevBrowserSTART(wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("*** START DEV BROWSER ***")

	a.CreateBrowserContext()
	// defer cancel()

	// crea un mapa para registrar los mensajes de log únicos
	uniqueLogs := make(map[string]bool)

	// captura los logs de JavaScript
	chromedp.ListenTarget(a.Context, func(ev interface{}) {
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
					// fmt.Printf("LOG: %s\n", s)
				}
			}
		}
	})

	// Navega a una página web
	err := chromedp.Run(a.Context, chromedp.Navigate("http://localhost:1234/level_3.html"))
	if err != nil {
		log.Fatal("Error al navegar "+a.Path+" ", err)
	}

	// Espera hasta que la página esté completamente cargada
	var loaded bool
	err = chromedp.Run(a.Context, chromedp.ActionFunc(func(ctx context.Context) error {
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
