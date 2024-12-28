package godev

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (h *handler) StartProgram() {
	// Verificar si la terminal está lista
	if h.terminal == nil || h.tea == nil {
		h.NewTerminal()
	}

	// Agregar mensaje inicial
	h.terminal.messages = append(h.terminal.messages,
		fmt.Sprintf("%s: Starting program...", time.Now().Format("15:04:05")))

	if h.tea != nil {
		h.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		time.Sleep(100 * time.Millisecond)
	}

	// BUILD AND RUN
	err := h.buildAndRun()
	if err != nil {
		h.terminal.messages = append(h.terminal.messages,
			fmt.Sprintf("%s: Error - %s", time.Now().Format("15:04:05"), err.Error()))
		h.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		time.Sleep(100 * time.Millisecond)
	}
}

func (h *handler) Restart(event_name string) error {
	var this = errors.New("Restart error")
	fmt.Println("Restarting APP..." + event_name)

	// STOP
	err := h.StopProgram()
	if err != nil {
		return errors.Join(this, errors.New("when closing app"), err)
	}

	// BUILD AND RUN
	err = h.buildAndRun()
	if err != nil {
		return errors.Join(this, errors.New("when building and starting app"), err)
	}

	return nil
}

func (h *handler) buildAndRun() (err error) {
	var this = errors.New("buildAndRun")
	h.terminal.PrintWarning(fmt.Sprintf("Building and Running %s...", h.main_file))

	// Eliminar el ejecutable anterior si existe
	if _, err := os.Stat(h.main_file); err == nil {
		err := os.Remove(h.main_file)
		if err != nil {
			return errors.Join(this, err)
		}
	}
	// flags, err := ldflags.Add(
	// 	d.TwoKeys.GetTwoPublicKeysWasmClientAndGoServer(),
	// // sessionbackend.AddPrivateSecretKeySigning(),
	// )

	// var ldflags = `-X 'main.version=` + tag + `'`

	// Construir el comando de compilación con el archivo correcto
	// Construir el comando de compilación con el archivo correcto
	buildCmd := []string{"go", "build", "-o", path.Join(h.output_dir, h.output_name)}
	
	// Si el archivo está en otro directorio, cambiar al directorio primero
	fileDir := path.Dir(h.main_file)
	fileName := path.Base(h.main_file)
	
	if fileDir != "." {
		buildCmd = append(buildCmd, fileName)
		h.Cmd = exec.Command(buildCmd[0], buildCmd[1:]...)
		h.Cmd.Dir = fileDir
	} else {
		h.Cmd = exec.Command(buildCmd[0], buildCmd[1:]...)
	}
	// h.Cmd = exec.Command("go", "build", "-o", h.app_path, "-ldflags", flags, "main.go")
	// d.Cmd = exec.Command("go", "build", "-o", d.app_path, "main.go" )

	stderr, er := h.Cmd.StderrPipe()
	if er != nil {
		return errors.Join(this, err)
	}

	stdout, er := h.Cmd.StdoutPipe()
	if er != nil {
		return errors.Join(this, err)
	}

	er = h.Cmd.Start()
	if er != nil {
		return errors.Join(this, err)
	}

	go io.Copy(h, stdout)
	errBuf, _ := io.ReadAll(stderr)

	// Esperar
	er = h.Cmd.Wait()
	if er != nil {
		return errors.Join(this, errors.New(string(errBuf)), err)
	}

	return h.run()
}

// Construir el comando con argumentos dinámicos
// cmdArgs := append([]string{"go", "build", "-o", d.app_path, "main.go"}, os.Args...)
// d.Cmd = exec.Command(cmdArgs[0], cmdArgs[1:]...)

func (h *handler) run() error {
	var this = errors.New("run")

	h.Cmd = exec.Command(h.main_file, h.run_arguments...)

	stderr, err := h.Cmd.StderrPipe()
	if err != nil {
		return errors.Join(this, err)
	}

	stdout, err := h.Cmd.StdoutPipe()
	if err != nil {
		return errors.Join(this, err)
	}

	err = h.Cmd.Start()
	if err != nil {
		return errors.Join(this, err)
	}

	// Capturar salida estándar y de error
	go func() {
		_, err := io.Copy(h, stdout)
		if err != nil {
			h.terminal.PrintError("Error capturing stdout:", err.Error())
		}
	}()

	go func() {
		_, err := io.Copy(h, stderr)
		if err != nil {
			h.terminal.PrintError("Error capturing stderr:", err.Error())
		}
	}()

	return nil
}

func (h handler) Write(p []byte) (n int, err error) {
	msg := string(p)

	// Limpiar y formatear el mensaje
	msg = strings.TrimSpace(msg)
	if msg != "" {
		// Agregar el mensaje con timestamp
		timestamp := time.Now().Format("15:04:05")
		formattedMsg := fmt.Sprintf("[%s] %s", timestamp, msg)

		// Agregar el mensaje al terminal
		if h.terminal != nil {
			h.terminal.messages = append(h.terminal.messages, formattedMsg)

			// Forzar actualización de la terminal
			if h.tea != nil {
				h.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
				time.Sleep(100 * time.Millisecond) // Dar tiempo para mostrar el mensaje
			}
		} else {
			// Si no hay terminal, imprimir directamente
			fmt.Print(formattedMsg)
		}
	}

	return len(p), nil
}

func (h *handler) StopProgram() error {
	if h.Cmd == nil || h.Cmd.Process == nil {
		return errors.New("no running process to stop")
	}

	pid := h.Cmd.Process.Pid
	h.terminal.PrintWarning(fmt.Sprintf("Stopping app PID %d", pid))

	// Enviar mensaje de cierre directamente al terminal
	if h.terminal != nil {
		h.terminal.messages = append(h.terminal.messages,
			fmt.Sprintf("%s: Stopping program...", time.Now().Format("15:04:05")))
	}

	// Forzar actualización de la terminal
	if h.tea != nil {
		h.tea.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		time.Sleep(500 * time.Millisecond) // Dar tiempo para mostrar el mensaje
	}

	return h.Cmd.Process.Kill()
}
