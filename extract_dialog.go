package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"./ffmpeg"
	"./logger"
	"github.com/cheggaaa/pb/v3"
	"github.com/manifoldco/promptui"
)

const (

	// Format for SRT timestamps.
	timestampFormat = "15:04:05.000"
)

var (
	srtTimingRegex = regexp.MustCompile(`^([^,]+),([^ ]+) --> ([^,]+),([^ ]+)$`)
	videoPathRegex = regexp.MustCompile(`.*/([^/]+).mkv`)

	progressBarTemplate = `{{string . "current_action" | green}} {{ bar . "[" "▮" (cycle . "▮" ) "_"}} {{percent .}}`

	// Threshold for trimming a gap between dialog segments.
	threshold, _ = time.ParseDuration("1.5s")

	// Logging.
	logPath = "./log.txt"
	l       = logger.New(&logPath)

	supportedFormats = []string{"mkv"}
)

func main() {
	inputPath := os.Args[1]

	isDir, err := isDirectory(inputPath)
	if err != nil {
		l.Fatal(err)
	}

	if isDir {
		processFolder(inputPath)
	} else {
		processOneFile(inputPath)
	}

	// Write to log file.
	l.WriteToFile()
}

func processFolder(folderPath string) {
	files, err := ioutil.ReadDir(folderPath)
	if err != nil {
		log.Fatal(err)
	}

	// var paths []string
	// for _, f := range files {
	// 	cur := filepath.Join(folderPath, f.Name())
	// 	for _, ext := range supportedFormats {
	// 		if strings.HasSuffix(cur, "."+ext) {
	// 			paths = append(paths, cur)
	// 		}
	// 	}
	// }
	var videos []*ffmpeg.Video
	for _, f := range files {
		cur := filepath.Join(folderPath, f.Name())
		for _, ext := range supportedFormats {
			if strings.HasSuffix(cur, "."+ext) {
				videos = append(videos, ffmpeg.NewVideo(l, cur))
			}
		}
	}

	sum := buildChapterSummary(videos)
	selectChapterSummary(sum)
}

func buildChapterSummary(videos []*ffmpeg.Video) []chapterSummary {
	m := make(map[string]int)
	sTimes := make(map[string][]string)
	eTimes := make(map[string][]string)

	for _, v := range videos {
		s, err := v.InfoStruct()
		if err != nil {
			l.Fatal(err)
		}

		for _, c := range s.Chapters {
			ivl := toInterval(c)
			m[ivl.Title] = m[ivl.Title] + 1
			sTimes[ivl.Title] = append(sTimes[ivl.Title], ivl.Start)
			eTimes[ivl.Title] = append(eTimes[ivl.Title], ivl.End)
		}
	}

	var out []chapterSummary
	for k := range m {
		starts := getMedian(sTimes[k])
		ends := getMedian(eTimes[k])
		out = append(out, chapterSummary{
			Title:       k,
			Count:       m[k],
			MedianStart: starts,
			MedianEnd:   ends,
		})
	}
	// Sort by median start time.
	sort.Slice(out, func(i, j int) bool {
		return out[i].MedianStart < out[j].MedianStart
	})
	return out
}

func getMedian(s []string) string {
	sort.Slice(s, func(i, j int) bool {
		return s[i] < s[j]
	})
	return s[len(s)/2]
}

type chapterSummary struct {
	Title       string
	Count       int
	MedianStart string
	MedianEnd   string
}

func selectChapterSummary(cs []chapterSummary) []chapterSummary {
	type wrapped struct {
		Checkbox    string
		Description string
		Summary     chapterSummary
	}
	w := []wrapped{{Checkbox: "✅ DONE", Description: ""}}

	checked := "✔️"
	unchecked := " "
	for _, st := range cs {
		w = append(w, wrapped{
			Summary:     st,
			Checkbox:    unchecked,
			Description: fmt.Sprintf("(%s - %s) found in %d videos", st.MedianStart, st.MedianEnd, st.Count),
		})
	}
	templates := &promptui.SelectTemplates{
		Label: `{{ . }}`,
		Active: `🌶 {{.Checkbox | green}} {{ .Summary.Title | cyan }}	{{ .Description | faint }}`,
		Inactive: `  {{.Checkbox | green}} {{ .Summary.Title | cyan }}	{{ .Description | faint }}`,
		Details: `
--------- Audio Track ----------
{{ "Title:" | faint }}	{{ .Summary.Title }}
{{ "Median time:" | faint }}	{{ .Summary.MedianStart | faint }} - {{ .Summary.MedianEnd | faint }}
{{ "Present in videos:" | faint }}	{{ .Summary.Count }}`,
	}
	pos := 1

	for true {
		prompt := promptui.Select{
			Label:        "Select chapter titles to ignore in all videos:",
			Items:        w,
			Templates:    templates,
			Size:         10,
			HideSelected: true,
			CursorPos:    pos,
		}

		choice, _, err := prompt.Run()
		if err != nil {
			l.Fatal(err)
		}

		if choice == 0 {
			break
		} else {
			pos = choice
			if w[choice].Checkbox == checked {
				w[choice].Checkbox = unchecked
			} else {
				w[choice].Checkbox = checked
			}
		}
	}

	var out []chapterSummary
	for _, i := range w {
		if i.Checkbox == checked {
			out = append(out, i.Summary)
		}
	}
	return out
}

func selectAudioTrack(s []ffmpeg.Stream, promptMessage, selectedLabel string) ffmpeg.Stream {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "\U0001F336 {{ .Tags.Title | cyan }} ({{ .Tags.Language | red }})",
		Inactive: "  {{ .Tags.Title | cyan }} ({{ .Tags.Language | red }})",
		Selected: selectedLabel + " {{ .Tags.Title | red | cyan }}",
		Details: `
--------- Audio Track ----------
{{ "Name:" | faint }}	{{ .Tags.Title }}
{{ "Language:" | faint }}	{{ .Tags.Language }}
{{ "Codec:" | faint }}	{{ .CodecLongName }}`,
	}

	prompt := promptui.Select{
		Label:     promptMessage,
		Items:     s,
		Templates: templates,
		Size:      4,
	}

	choice, _, err := prompt.Run()
	if err != nil {
		l.Fatal(err)
	}
	return s[choice]
}

func processOneFile(vidPath string) {
	v := ffmpeg.NewVideo(l, vidPath)
	err := v.LogFullFileInfo()
	if err != nil {
		l.Fatal(err.Error())
	}

	c := &ffmpeg.Configuration{
		TempDir:   "./.tmp/",
		OutputDir: "./out/",
	}

	// Create directories if needed.
	if _, err := os.Stat(c.TempDir); os.IsNotExist(err) {
		os.Mkdir(c.TempDir, 0755)
	}
	if _, err := os.Stat(c.OutputDir); os.IsNotExist(err) {
		os.Mkdir(c.OutputDir, 0755)
	}

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
		c.Audio = selectAudioTrack(s, "Select the audio track to use:", "Selected audio track:")
	}

	s, err = v.GetSubtitleStreams()
	if err != nil {
		l.Fatal(err.Error())
	}
	l.Println()
	if len(s) == 0 {
		l.Fatal("no subtitle tracks found")
	} else if len(s) == 1 {
		l.Printlnf("Found one subtitle track: %s (%s)", s[0].Tags.Title, s[0].Tags.Language)
		c.Subtitles = s[0]
	} else {
		templates := &promptui.SelectTemplates{
			Label:    "{{ . }}",
			Active:   "\U0001F336 {{ .Tags.Title | cyan }} ({{ .Tags.Language | red }})",
			Inactive: "  {{ .Tags.Title | cyan }} ({{ .Tags.Language | red }})",
			Selected: "Selected subtitle track: {{ .Tags.Title | red | cyan }}",
			Details: `
--------- Subtitle Track ----------
{{ "Name:" | faint }}	{{ .Tags.Title }}
{{ "Codec:" | faint }}	{{ .CodecLongName }}`,
		}

		prompt := promptui.Select{
			Label:     "Select the subtitle track to use:",
			Items:     s,
			Templates: templates,
			Size:      4,
		}

		choice, _, err := prompt.Run()
		if err != nil {
			l.Fatal(err)
		}
		c.Subtitles = s[choice]
	}

	l.Println()
	info, err := v.InfoStruct()
	if err != nil {
		l.Fatal(err.Error())
	}
	if len(info.Chapters) == 0 {
		l.Print("no chapters found, skipping step")
	} else {
		l.Println("This video has labeled chapters:")
		for i := 0; i < len(info.Chapters); i++ {
			cur := info.Chapters[i]
			ivl := toInterval(cur)
			l.Printlnf("\tOption %d: %s\t(%s - %s)", i, cur.Tags.Title, ivl.Start, ivl.End)
		}
		choices := requestMultipleInts("Choose chapters that should be ignored (comma-separated): ", 0, len(info.Chapters)-1)
		var chaps []ffmpeg.Chapter
		for i := 0; i < len(choices); i++ {
			chaps = append(chaps, info.Chapters[choices[i]])
		}
		c.SkippedChapters = chaps
	}

	extractDialog(v, *c)
}

// func selectMultipleChapters(options []ffmpeg.Chapter, message string) []ffmpeg.Chapter {
// 	var chosen []ffmpeg.Chapter

// 	for true {
// 		var intervals []ffmpeg.Interval
// 		for _, c := range options {
// 			intervals = append(intervals, toInterval(c))
// 		}

// 		templates := &promptui.SelectTemplates{
// 			Label:    "{{ . }}",
// 			Active:   "\U0001F336 {{ .Title | cyan }} ({{ .Start | red }} - {{ .End | red }})",
// 			Inactive: "  {{ .Tags.Title | cyan }} ({{ .Tags.Language | red }})",
// 			Selected: "Removed chapter: {{ .Tags.Title | red | cyan }}",
// 			Details: `
// 	--------- Subtitle Track ----------
// 	{{ "Name:" | faint }}	{{ .Tags.Title }}
// 	{{ "Codec:" | faint }}	{{ .CodecLongName }}`,
// 		}

// 		searcher := func(input string, index int) bool {
// 			pepper := intervals[index]
// 			name := strings.Replace(strings.ToLower(pepper.Title), " ", "", -1)
// 			input = strings.Replace(strings.ToLower(input), " ", "", -1)

// 			return strings.Contains(name, input)
// 		}

// 		prompt := promptui.Select{
// 			Label:     message,
// 			Items:     intervals,
// 			Templates: templates,
// 			Size:      4,
// 			Searcher:  searcher,
// 		}

// 		choice, _, err := prompt.Run()
// 		if err != nil {
// 			l.Fatal(err)
// 		}

// 		if choice == 0 {
// 			break
// 		} else {
// 			chosen = append(chosen)
// 		}
// 	}

// }

func extractDialog(v *ffmpeg.Video, c ffmpeg.Configuration) {
	_, err := v.ExtractSubtitles(c)
	if err != nil {
		return
	}

	comb := readAndCombineSubtitles(c.TempDir + "subs.srt")
	comb = subtractChapters(comb, c.SkippedChapters)

	// Create progress bar.
	bar := pb.ProgressBarTemplate(progressBarTemplate).Start(len(comb) + 3)

	bar.Set("current_action", "Copying audio")
	_, err = v.ExtractAudio(c)
	bar.Increment()

	outFile := ""
	for i := 0; i < len(comb); i++ {
		cur := comb[i]
		fname := "shard-" + fmt.Sprint(i) + ".mp3"
		outFile = outFile + "file '" + fname + "'" + "\n"
		bar.Set("current_action", fmt.Sprintf("Splitting audio (%d/%d)", i+1, len(comb)))
		_, err = v.ExtractAudioFromInterval(c, cur, c.TempDir+fname)
		if err != nil {
			return
		}
		bar.Increment()
	}

	// Write all fragment filenames to a text file.
	if err := ioutil.WriteFile(c.TempDir+"output.txt", []byte(outFile), 0644); err != nil {
		l.Fatal(err.Error())
	}

	// Combine all fragments into one file.
	bar.Set("current_action", "Joining audio fragments")
	audioOutPath := videoPathRegex.ReplaceAllString(v.Path, `$1.mp3`)
	if _, err = v.CatenateAudioFiles(c, c.TempDir+audioOutPath); err != nil {
		l.Fatal(err.Error())
	}
	bar.Increment()

	// Re-encode output file to repair any errors from catenation.
	bar.Set("current_action", "Re-encoding audio")
	if _, err = v.ReEncodeAudio(c, c.TempDir+audioOutPath, c.OutputDir+audioOutPath); err != nil {
		l.Fatal(err.Error())
	}
	bar.Increment()
	bar.Finish()

	// Delete temp dir.
	os.RemoveAll(c.TempDir)

	l.Printlnf("Action completed. Created file %s", c.TempDir+audioOutPath)
}

func subtractChapters(intervals []ffmpeg.Interval, chapters []ffmpeg.Chapter) []ffmpeg.Interval {
	if len(chapters) == 0 {
		return intervals
	}

	wip := intervals

	for j := 0; j < len(chapters); j++ {
		var rev []ffmpeg.Interval
		chap := toInterval(chapters[j])
		for i := 0; i < len(wip); i++ {
			cur := wip[i]
			if cur.Start > chap.Start && cur.Start < chap.End {
				cur = ffmpeg.Interval{
					Start: chap.End,
					End:   cur.End,
				}
			}
			if cur.End > chap.Start && cur.End < chap.End {
				cur = ffmpeg.Interval{
					Start: cur.Start,
					End:   chap.Start,
				}
			}
			if cur.Start < cur.End {
				rev = append(rev, cur)
			}
		}
		wip = rev
	}
	return wip
}

func readAndCombineSubtitles(subPath string) []ffmpeg.Interval {
	file, err := os.Open(subPath)
	if err != nil {
		l.Fatal(err.Error())
	}
	defer file.Close()

	var readIn []ffmpeg.Interval
	scanner := bufio.NewScanner(file)
	i := 0
	for scanner.Scan() {
		l := scanner.Text()
		if strings.Contains(l, "-->") {
			start := srtTimingRegex.ReplaceAllString(l, `$1.$2`)
			end := srtTimingRegex.ReplaceAllString(l, `$3.$4`)
			readIn = append(readIn, ffmpeg.Interval{Start: start, End: end})
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
func combineIntervals(intervals []ffmpeg.Interval, gapThreshold time.Duration) []ffmpeg.Interval {
	if len(intervals) == 0 {
		l.Fatal("No subtitles were found in the file. Aborting.")
	}
	// Sort by start time.
	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i].Start < intervals[j].Start
	})

	var combined []ffmpeg.Interval
	pending := intervals[0]
	for i := 1; i < len(intervals); i++ {
		cur := intervals[i]
		if cur.Start < pending.End || !isGapOverThreshold(pending.End, cur.Start, gapThreshold) {
			if cur.End >= pending.End {
				pending = ffmpeg.Interval{Start: pending.Start, End: cur.End}
			}
		} else {
			if pending.Start != pending.End {
				combined = append(combined, pending)
			}
			pending = cur
		}
	}
	if pending.Start != pending.End {
		combined = append(combined, pending)
	}
	return combined
}

func toInterval(chapter ffmpeg.Chapter) ffmpeg.Interval {
	start, _ := time.ParseDuration(chapter.StartTime + "s")
	end, _ := time.ParseDuration(chapter.EndTime + "s")

	zero, _ := time.Parse("00:00:00,000", timestampFormat)
	return ffmpeg.Interval{
		Start: zero.Add(start).Format(timestampFormat),
		End:   zero.Add(end).Format(timestampFormat),
		Title: chapter.Tags.Title,
	}
}

// requestInt asks the user for a bounded number using stdio.
func requestInt(message string, min, max int) int {
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

// requestInt asks the user for a bounded number using stdio.
func requestMultipleInts(message string, min, max int) []int {
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

func isDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	l.Printlnf("Hey you %s", fileInfo.Name())
	if err != nil {
		return false, err
	}
	return fileInfo.IsDir(), err
}
