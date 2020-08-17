package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"./logger"
)

const (
	tempDir = "./tmp/"

	timestampFormat = "15:04:05.000"
)

var (
	re        = regexp.MustCompile(`^([^,]+),([^ ]+) --> ([^,]+),([^ ]+)$`)
	logOutput = ""
	// Threshold between dialog that will result in the silence being trimmed.
	threshold, _ = time.ParseDuration("1s")
	logPath      = "./log.txt"
	l            = logger.New(&logPath)
)

func runShellCommand(name string, arg ...string) ([]byte, error) {
	cmd := exec.Command(name, arg...)
	l.Println("Executing command: " + cmd.String())
	out, err := cmd.CombinedOutput()
	if err != nil {
		l.Println(err.Error())
		l.Println("Command output: " + string(out))
	}

	return out, err
}

// Interval represents a time interval over which subtitles are displayed.
type Interval struct {
	start string
	end   string
}

func main() {
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

	// Write all fragment filenames to a text file.
	if err := ioutil.WriteFile(tempDir+"output.txt", []byte(outFile), 0644); err != nil {
		l.Fatal(err.Error())
	}

	// Combine all fragments into one file.
	if _, err = runShellCommand("ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", tempDir+"output.txt", "-c", "copy", tempDir+audioOutPath); err != nil {
		l.Fatal(err.Error())
	}

	// Re-encode output file to repair any errors from catenation.
	if _, err = runShellCommand("ffmpeg", "-y", "-i", tempDir+audioOutPath, "-c:v", "copy", audioOutPath); err != nil {
		l.Fatal(err.Error())
	}

	// Delete temp dir.
	os.RemoveAll(tempDir)

	// Write to log file.
	l.WriteToFile()

	l.Println("Action complete.")
}

func readAndCombineSubtitles(subPath string) []Interval {
	file, err := os.Open(subPath)
	if err != nil {
		l.Fatal(err.Error())
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
		l.Fatal(err.Error())
	}

	return combineIntervals(readIn, threshold)
}

// gapOverThreshold decides if a gap between two points is over a duration threshold.
func gapOverThreshold(start, end string, gapThreshold time.Duration) bool {
	startTime, err := time.Parse(timestampFormat, start)
	if err != nil {
		l.Fatal(err.Error())
	}
	endTime, err := time.Parse(timestampFormat, end)
	if err != nil {
		l.Fatal(err.Error())
	}
	if endTime.After(startTime) {
		return endTime.Sub(startTime) > gapThreshold
	}
	return startTime.Sub(endTime) > gapThreshold
}

// combineIntervals combines possibly overlapping intervals and de-dupes and combines them when necessary.
func combineIntervals(intervals []Interval, gapThreshold time.Duration) []Interval {
	if len(intervals) <= 2 {
		l.Fatal("Less than three subtitles were found in the file. Aborting.")
	}
	// Sort by start time.
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})

	var combined []Interval
	pending := intervals[0]
	for i := 1; i < len(intervals); i++ {
		cur := intervals[i]
		if cur.start < pending.end || !gapOverThreshold(pending.end, cur.start, gapThreshold) {
			if cur.end >= pending.end {
				pending = Interval{start: pending.start, end: cur.end}
			}
		} else {
			if pending.start != pending.end {
				combined = append(combined, pending)
			}
			pending = cur
		}
	}
	if pending.start != pending.end {
		combined = append(combined, pending)
	}
	return combined
}
