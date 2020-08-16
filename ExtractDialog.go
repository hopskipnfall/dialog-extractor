package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

const (
	tempDir = "./tmp/"
)

var re = regexp.MustCompile(`^([^,]+),([^ ]+) --> ([^,]+),([^ ]+)$`)

func runShellCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	fmt.Println("Executing command: " + cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Print("The output" + string(out))
	return out, err
}

func main() {
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, 0755)
	}

	// First parameter
	vidPath := os.Args[1]

	_, err := runShellCommand("ffmpeg", "-i", vidPath, "-map", "0:s:0", tempDir+"subs.srt")
	if err != nil {
		return
	}

	file, err := os.Open(tempDir + "subs.srt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	outFile := ""

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		l := scanner.Text()
		if strings.Contains(l, "-->") {
			fname := "file-" + fmt.Sprint(i) + ".aac"
			outFile = outFile + "file '" + fname + "'" + "\n"
			start := re.ReplaceAllString(l, `$1.$2`)
			stop := re.ReplaceAllString(l, `$3.$4`)
			_, err = runShellCommand("ffmpeg", "-i", vidPath, "-ss", start, "-to", stop, "-c", "copy", tempDir+fname)
			if err != nil {
				return
			}
			i = i + 1
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	err = ioutil.WriteFile(tempDir+"output.txt", []byte(outFile), 0644)
	if err != nil {
		panic(err)
	}

	_, err = runShellCommand("ffmpeg", "-f", "concat", "-safe", "0", "-i", tempDir+"output.txt", "-c", "copy", "combined.aac")
	if err != nil {
		return
	}

	os.RemoveAll(tempDir)
}
