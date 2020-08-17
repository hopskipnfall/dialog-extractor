package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	tempDir = "./tmp/"
	logPath = "./log.txt"

	timestampFormat = "15:04:05.000"
)

var (
	re        = regexp.MustCompile(`^([^,]+),([^ ]+) --> ([^,]+),([^ ]+)$`)
	logOutput = ""
	// Threshold between dialog that will result in the silence being trimmed.
	threshold, _ = time.ParseDuration("1s")
)

func print(message string) {
	logOutput = logOutput + message
	fmt.Println(message)
}

func newline() {
	logOutput = logOutput + "\n"
	fmt.Println()
}

func println(message string) {
	print(message)
	newline()
}

func printlnf(format string, a ...interface{}) {
	println(fmt.Sprintf(format, a...))
}

func printf(format string, a ...interface{}) {
	print(fmt.Sprintf(format, a...))
}

func runShellCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	println("Executing command: " + cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		println(err.Error())
		println("Command output: " + string(out))
	}

	return out, err
}

// Interval represents a time interval over which subtitles are displayed.
type Interval struct {
	start string
	end   string
}

func readAndCombineSubtitles(subPath string) []Interval {
	file, err := os.Open(subPath)
	if err != nil {
		println(err.Error())
		log.Fatal(err)
	}
	defer file.Close()

	var readIn []Interval
	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		l := scanner.Text()
		if strings.Contains(l, "-->") {
			start := re.ReplaceAllString(l, `$1.$2`)
			end := re.ReplaceAllString(l, `$3.$4`)
			readIn = append(readIn, Interval{start: start, end: end})
			i = i + 1
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return combineIntervals(readIn, threshold)
}

func gapOverThreshold(start, end string, gapThreshold time.Duration) bool {
	startTime, err := time.Parse(timestampFormat, start)
	if err != nil {
		panic(err)
	}
	endTime, err := time.Parse(timestampFormat, end)
	if err != nil {
		panic(err)
	}
	if endTime.After(startTime) {
		return endTime.Sub(startTime) > gapThreshold
	}
	return startTime.Sub(endTime) > gapThreshold
}

func combineIntervals(intervals []Interval, gapThreshold time.Duration) []Interval {
	if len(intervals) == 0 {
		println("No subtitles found!")
		log.Fatal("No subtitles found!")
	}
	// Sort.
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})

	var combined []Interval
	pending := intervals[0]
	printlnf("\n%s - %s  START\n", pending.start, pending.end)
	for i := 1; i < len(intervals); i++ {
		cur := intervals[i]
		printf("%s - %s  ", cur.start, cur.end)
		if cur.start < pending.end || !gapOverThreshold(pending.end, cur.start, gapThreshold) {
			if cur.end >= pending.end {
				print("overlap")
				pending = Interval{start: pending.start, end: cur.end}
			}
		} else {
			printf("              appended %s - %s  ", pending.start, pending.end)
			if pending.start != pending.end {
				combined = append(combined, pending)
			}
			pending = cur
		}
		newline()
	}
	if pending.start != pending.end {
		combined = append(combined, pending)
	}
	return combined
}

func main() {
	logOutput = ""
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, 0755)
	}

	// First parameter
	vidPath := os.Args[1]

	re2, _ := regexp.Compile(`.*/([^/]+).mkv`)
	audioOutPath := re2.ReplaceAllString(vidPath, `$1.mp3`)

	_, err := runShellCommand("ffmpeg", "-y", "-i", vidPath, "-map", "0:s:0", tempDir+"subs.srt")
	if err != nil {
		return
	}

	comb := readAndCombineSubtitles(tempDir + "subs.srt")

	mp3ScratchPath := tempDir + "full_audio.mp3"
	_, err = runShellCommand("ffmpeg", "-y", "-i", vidPath, "-q:a", "0", "-map", "a", mp3ScratchPath)

	outFile := ""
	for i := 0; i < len(comb); i++ {
		cur := comb[i]
		fname := "file-" + fmt.Sprint(i) + ".mp3"
		outFile = outFile + "file '" + fname + "'" + "\n"
		_, err = runShellCommand("ffmpeg", "-y", "-i", mp3ScratchPath, "-ss", cur.start, "-to", cur.end, "-q:a", "0", "-map", "a", tempDir+fname)
		if err != nil {
			return
		}
	}

	if err := ioutil.WriteFile(tempDir+"output.txt", []byte(outFile), 0644); err != nil {
		panic(err)
	}

	if _, err = runShellCommand("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", tempDir+"output.txt", "-c", "copy", audioOutPath); err != nil {
		panic(err)
	}

	os.RemoveAll(tempDir)

	if err := ioutil.WriteFile(logPath, []byte(logOutput), 0644); err != nil {
		panic(err)
	}

	println("Action complete.")
}
