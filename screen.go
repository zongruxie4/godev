package godev

import (
	"fmt"

	"github.com/fstanis/screenresolution"
)

func GetScreenSize() (int, int, error) {

	res := screenresolution.GetPrimary()
	if res == nil {
		return 0, 0, fmt.Errorf("error GetScreenSize sistema operativo no soportado")
	} else {

		return res.Width, res.Height, nil
	}

}
