package shell

import (
	"bufio"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"../logger"
)

// ExecuteCommand runs a shell command and logs the result only if an error occurred.
func ExecuteCommand(l *logger.Logger, name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	// l.Println("Executing command: " + cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		l.Printlnf("Failed executing command: %s\n with error: %v", cmd.String(), err)
		l.Println("Output: " + string(out))
	}

	return out, err
}

// RequestInt asks the user for a bounded number using stdio.
func RequestInt(l *logger.Logger, message string, min, max int) int {
	for true {
		l.Print(message)
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		choice, err := strconv.Atoi(text)
		if err != nil || choice < min || choice > max {
			l.Println("illegal choice, try again.")
		} else {
			return choice
		}
	}
	panic("this is impossible")
}

// RequestMultipleInts asks the user for a bounded number using stdio.
func RequestMultipleInts(l *logger.Logger, message string, min, max int) []int {
	for true {
		l.Print(message)
		reader := bufio.NewReader(os.Stdin)
		text, _ := reader.ReadString('\n')
		if text == "" {
			return []int{}
		}

		var out []int
		valid := true
		choices := strings.Split(text, ",")
		for i := 0; i < len(choices); i++ {
			choice, err := strconv.Atoi(strings.TrimSpace(choices[i]))
			if err != nil || choice < min || choice > max {
				l.Println("illegal choice, try again.")
				valid = false
			} else {
				out = append(out, choice)
			}
		}
		if valid {
			return out
		}
	}
	panic("this is impossible")
}
