package golite

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestInstallScript(t *testing.T) {

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// En Windows, intentamos primero con Git Bash
		gitBashPath := "C:\\Program Files\\Git\\bin\\bash.exe"
		if _, err := os.Stat(gitBashPath); err == nil {
			cmd = exec.Command(gitBashPath, "-c", "./install.sh")
		} else {
			// Si no encuentra Git Bash, intenta con WSL
			cmd = exec.Command("wsl", "./install.sh")
		}
	default:
		// Para Linux y macOS
		cmd = exec.Command("bash", "./install.sh")
	}

	// Configurar el directorio de trabajo
	cmd.Dir = "." // Asegúrate de que esto apunta al directorio correcto

	// Capturar tanto stdout como stderr
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("Failed to execute install.sh: %v\nOutput: %s", err, string(output))
	}
}
