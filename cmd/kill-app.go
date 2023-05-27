package main

import "syscall"

func killApp() error {
	if appProcess != nil {
		err := appProcess.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}
	}

	return nil
}
