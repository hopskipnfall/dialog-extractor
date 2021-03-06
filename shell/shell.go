package shell

import (
	"os/exec"

	"../logger"
)

// ExecuteCommand runs a shell command and logs the result only if an error occurred.
func ExecuteCommand(l *logger.Logger, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	// l.Println("Executing command: " + cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		l.Printlnf("Command failed with error: %v", err)
		l.Println("Output: " + string(out))
	}

	return out, err
}
