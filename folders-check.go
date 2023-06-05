package godev

import (
	"fmt"
	"log"
	"os"
)

func (u ui) checkStaticFileFolders() {
	dirs := []string{u.FolderPath() + "/static", u.build_folder + "/static", "modules"}

	for _, dir := range dirs {
		_, err := os.Stat(dir)
		if os.IsNotExist(err) {
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				fmt.Printf("Error creando directorio %s: %v", dir, err)
				return
			}
			fmt.Printf("Directorio %s creado correctamente.\n", dir)
		} else if err != nil {
			fmt.Printf("Error al verificar directorio %s: %v", dir, err)
			return
		}
		// else {
		// 	fmt.Printf("Directorio %s ya existe.\n", dir)
		// }
	}
}

func isDir(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		log.Println(err)
		return false
	}
	return fi.Mode().IsDir()
}
