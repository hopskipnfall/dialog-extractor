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

	vidPath := "./Hataraku Maou-sama!/Hataraku Maou-sama! - 01.mkv"

	//ffmpeg -i 'Hataraku Maou-sama! - 01.mkv' -map 0:s:0 subs.srt
	runShellCommand("ffmpeg", "-i", vidPath, "-map", "0:s:0", tempDir+"subs.srt")

	file, err := os.Open("./out/subs.srt")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	outFile := ""

	// argsWithProg := os.Args
	// argsWithoutProg := os.Args[1:]
	// fmt.Println(argsWithProg)
	// fmt.Println(argsWithoutProg)
	// outFile = outFile + strings.Join(argsWithProg, ",") + "\n"
	// outFile = outFile + strings.Join(argsWithoutProg, ",") + "\n"

	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		l := scanner.Text()
		if strings.Contains(l, "-->") {
			fname := "./out/file-" + fmt.Sprint(i) + ".aac"
			outFile = outFile + "file '" + fname + "'" + "\n"
			start := re.ReplaceAllString(l, `$1.$2`)
			stop := re.ReplaceAllString(l, `$3.$4`)
			runShellCommand("ffmpeg", "-i", vidPath, "-ss", start, "-to", stop, "-c", "copy", fname)
			i = i + 1
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// write the whole body at once
	err = ioutil.WriteFile("out/output.txt", []byte(outFile), 0644)
	if err != nil {
		panic(err)
	}

	runShellCommand("ffmpeg", "-f", "concat", "-safe", "0", "-i", tempDir+"output.txt", "-c", "copy", "combined.aac")

	os.RemoveAll(tempDir)
}
