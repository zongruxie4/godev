package godev

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/cdvelop/git"
)

func (d Dev) AppVersion() string {

	tag, err := git.GetLatestTag()
	if err != "" {
		log.Println("AppVersion error", err)
	}

	return tag
}

func (d *Dev) AppName() string {
	const e = "AppName error "

	current_dir, err := os.Getwd()
	if err != nil {
		log.Println(e, err)
		return "app"
	}

	return filepath.Base(current_dir)

}

func (d Dev) AppFileName() string {
	const e = "AppFileName error "

	if runtime.GOOS == "windows" {
		return d.AppName() + "-" + d.AppVersion() + ".exe"
	}

	return d.AppName() + "-" + d.AppVersion()

}
