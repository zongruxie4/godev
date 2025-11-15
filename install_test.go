package golite

import (
	"os/exec"
	"runtime"
	"testing"
)

func TestInstallScript(t *testing.T) {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// En Windows, usa la ruta completa de Git Bash
		cmd = exec.Command("C:\\Users\\juan7\\AppData\\Local\\Programs\\Git\\bin\\bash.exe", "./install.sh")
	} else {
		// En Linux/Mac usa bash normal
		cmd = exec.Command("bash", "./install.sh")
	}

	// Capturar tanto stdout como stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Failed to execute install.sh: %v\nOutput: %s", err, string(output))
	}
}
