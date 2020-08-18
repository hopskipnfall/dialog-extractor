package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"./ffmpeg"
	"./logger"
	"./shell"
)

const (
	tempDir   = "./.tmp/"
	outputDir = "./out/"

	timestampFormat   = "15:04:05.000"
	ffmpegInputNumber = 0
)

var (
	srtTimingRegex = regexp.MustCompile(`^([^,]+),([^ ]+) --> ([^,]+),([^ ]+)$`)
	videoPathRegex = regexp.MustCompile(`.*/([^/]+).mkv`)

	// Threshold for trimming a gap between dialog segments.
	threshold, _ = time.ParseDuration("1.5s")

	// Logging.
	logPath = "./log.txt"
	l       = logger.New(&logPath)
)

type Configuration struct {
	Subtitles ffmpeg.Stream
	Audio     ffmpeg.Stream
}

func main() {
	// Create directories if needed.
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, 0755)
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.Mkdir(outputDir, 0755)
	}

	// Video path is the first argument.
	vidPath := os.Args[1]

	v := ffmpeg.NewVideo(l, vidPath)
	err := v.LogFullFileInfo()
	if err != nil {
		l.Fatal(err.Error())
	}

	c := &Configuration{}

	s, err := v.GetAudioStreams()
	if err != nil {
		l.Fatal(err.Error())
	}

	if len(s) == 0 {
		l.Fatal("no audio tracks found")
	} else if len(s) == 1 {
		l.Printlnf("Found one audio track: %s (%s)", s[0].Tags.Title, s[0].Tags.Language)
		c.Audio = s[0]
	} else {
		l.Println("Found multiple audio tracks:")
		for i := 0; i < len(s); i++ {
			cur := s[i]
			l.Printlnf("\t%d: Title: %s (%s)", i, cur.Tags.Title, cur.Tags.Language)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Choose number: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		choice, err := strconv.Atoi(text)
		if err != nil {
			l.Fatal("illegal choice")
		}
		c.Audio = s[choice]
	}

	s, err = v.GetSubtitleStreams()
	if err != nil {
		l.Fatal(err.Error())
	}

	if len(s) == 0 {
		l.Fatal("no subtitle tracks found")
	} else if len(s) == 1 {
		l.Printlnf("Found one subtitle track: %s (%s)", s[0].Tags.Title, s[0].Tags.Language)
		c.Subtitles = s[0]
	} else {
		l.Println("Found multiple subtitle tracks:")
		for i := 0; i < len(s); i++ {
			cur := s[i]
			l.Printlnf("\t%d: Title: %s (%s)", i, cur.Tags.Title, cur.Tags.Language)
		}
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Choose number: ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		choice, err := strconv.Atoi(text)
		if err != nil {
			l.Fatal("illegal choice")
		}
		c.Subtitles = s[choice]
	}

	processFile(vidPath, *c)

	// Write to log file.
	l.WriteToFile()
	l.Println("Action complete.")
}

func processFile(vidPath string, c Configuration) {
	audioOutPath := videoPathRegex.ReplaceAllString(vidPath, `$1.mp3`)

	_, err := shell.ExecuteCommand(l, "ffmpeg", "-y", "-i", vidPath, "-map", fmt.Sprintf("%d:%d", ffmpegInputNumber, c.Subtitles.Index), tempDir+"subs.srt")
	if err != nil {
		return
	}

	comb := readAndCombineSubtitles(tempDir + "subs.srt")

	mp3ScratchPath := tempDir + "full_audio.mp3"
	_, err = shell.ExecuteCommand(l, "ffmpeg", "-y", "-i", vidPath, "-q:a", "0", "-map", fmt.Sprintf("%d:%d", ffmpegInputNumber, c.Audio.Index), mp3ScratchPath)
	outFile := ""
	for i := 0; i < len(comb); i++ {
		cur := comb[i]
		fname := "shard-" + fmt.Sprint(i) + ".mp3"
		outFile = outFile + "file '" + fname + "'" + "\n"
		_, err = shell.ExecuteCommand(l, "ffmpeg", "-y", "-i", mp3ScratchPath, "-ss", cur.start, "-to", cur.end, "-q:a", "0", "-map", "a", tempDir+fname)
		if err != nil {
			return
		}
	}

	// Write all fragment filenames to a text file.
	if err := ioutil.WriteFile(tempDir+"output.txt", []byte(outFile), 0644); err != nil {
		l.Fatal(err.Error())
	}

	// Combine all fragments into one file.
	if _, err = shell.ExecuteCommand(l, "ffmpeg", "-y", "-f", "concat", "-safe", "0", "-i", tempDir+"output.txt", "-c", "copy", tempDir+audioOutPath); err != nil {
		l.Fatal(err.Error())
	}

	// Re-encode output file to repair any errors from catenation.
	if _, err = shell.ExecuteCommand(l, "ffmpeg", "-y", "-i", tempDir+audioOutPath, "-c:v", "copy", outputDir+audioOutPath); err != nil {
		l.Fatal(err.Error())
	}

	// Delete temp dir.
	os.RemoveAll(tempDir)
}

// Interval represents a time interval over which subtitles are displayed.
type Interval struct {
	start string
	end   string
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
			start := srtTimingRegex.ReplaceAllString(l, `$1.$2`)
			end := srtTimingRegex.ReplaceAllString(l, `$3.$4`)
			readIn = append(readIn, Interval{start: start, end: end})
			i = i + 1
		}
	}
	if err := scanner.Err(); err != nil {
		l.Fatal(err.Error())
	}

	return combineIntervals(readIn, threshold)
}

// isGapOverThreshold decides if a gap between two points is over a duration threshold.
func isGapOverThreshold(start, end string, gapThreshold time.Duration) bool {
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
	if len(intervals) == 0 {
		l.Fatal("No subtitles were found in the file. Aborting.")
	}
	// Sort by start time.
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].start < intervals[j].start
	})

	var combined []Interval
	pending := intervals[0]
	for i := 1; i < len(intervals); i++ {
		cur := intervals[i]
		if cur.start < pending.end || !isGapOverThreshold(pending.end, cur.start, gapThreshold) {
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
