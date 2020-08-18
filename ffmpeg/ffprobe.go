package ffmpeg

import (
	"encoding/json"

	"../logger"
	"../shell"
)

type Disposition struct {
	Default         int `json:"default,omitempty"`
	Dub             int `json:"dub,omitempty"`
	Original        int `json:"original,omitempty"`
	Comment         int `json:"comment,omitempty"`
	Lyrics          int `json:"lyrics,omitempty"`
	Karaoke         int `json:"karaoke,omitempty"`
	Forced          int `json:"forced,omitempty"`
	HearingImpared  int `json:"hearing_impaired,omitempty"`
	VisualImpared   int `json:"visual_impaired,omitempty"`
	CleanEffects    int `json:"clean_effects,omitempty"`
	AttachedPic     int `json:"attached_pic,omitempty"`
	TimedThumbnails int `json:"timed_thumbnails,omitempty"`
}

type Stream struct {
	Index              int    `json:"index,omitempty"`
	CodecName          string `json:"codec_name,omitempty"`
	CodecLongName      string `json:"codec_long_name,omitempty"`
	Profile            string `json:"profile,omitempty"`
	CodecType          string `json:"codec_type,omitempty"`
	CodecTimeBase      string `json:"codec_time_base,omitempty"`
	CodecTagString     string `json:"codec_tag_string,omitempty"`
	CodecTag           string `json:"codec_tag,omitempty"`
	Width              int    `json:"width,omitempty"`
	Height             int    `json:"height,omitempty"`
	CodedWidth         int    `json:"coded_width,omitempty"`
	CodedHeight        int    `json:"coded_height,omitempty"`
	HasBFrames         int    `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string `json:"display_aspect_ratio,omitempty"`
	PixFmt             string `json:"pix_fmt,omitempty"`
	Level              int    `json:"level,omitempty"`
	ChromaLocation     string `json:"chroma_location,omitempty"`
	FieldOrder         string `json:"field_order,omitempty"`
	Refs               int    `json:"refs,omitempty"`
	IsAvc              string `json:"is_avc,omitempty"`
	NalLengthSize      string `json:"nal_length_size,omitempty"`
	RFrameRate         string `json:"r_frame_rate,omitempty"`
	AvgFrameRate       string `json:"avg_frame_rate,omitempty"`
	TimeBase           string `json:"time_base,omitempty"`
	StartPts           int    `json:"start_pts,omitempty"`
	StartTime          string `json:"start_time,omitempty"`
	BitsPerRawSample   string `json:"bits_per_raw_sample,omitempty"`

	Disposition Disposition `json:"disposition,omitempty"`
	Tags        Tags        `json:"tags,omitempty"`
}

type Tags struct {
	Title    string `json:"title,omitempty"`
	Language string `json:"language,omitempty"`
}

type Chapter struct {
	ID        int64  `json:"id,omitempty"`
	TimeBase  string `json:"time_base,omitempty"`
	Start     int64  `json:"start,omitempty"`
	StartTime string `json:"start_time,omitempty"`
	End       int64  `json:"end,omitempty"`
	EndTime   string `json:"end_time,omitempty"`

	Tags Tags `json:"tags,omitempty"`
}

type Video struct {
	l    *logger.Logger
	path string
}

// New Creates a new logger.
func NewVideo(logger *logger.Logger, path string) *Video {
	v := &Video{
		l:    logger,
		path: path,
	}
	return v
}

type VideoInfo struct {
	Streams  []Stream  `json:"streams"`
	Chapters []Chapter `json:"chapters"`
}

// FullFileInfo requests info from ffprobe in json form.
func (v *Video) LogFullFileInfo() error {
	res, err := shell.ExecuteCommand(v.l, "ffprobe", v.path)
	if err != nil {
		return err
	}
	v.l.Println(string(res))
	return nil
}

func (v *Video) InfoStruct() (*VideoInfo, error) {
	res, err := shell.ExecuteCommand(v.l, "ffprobe", v.path, "-show_streams", "-show_chapters", "-v", "quiet", "-print_format", "json")
	if err != nil {
		return nil, err
	}
	i := VideoInfo{}
	err = json.Unmarshal(res, &v)
	return &i, err
}

func (v *Video) GetAudioStreams() ([]Stream, error) {
	// TODO: Use InfoStruct instead.
	res, err := shell.ExecuteCommand(v.l, "ffprobe", v.path, "-select_streams", "a", "-show_streams", "-v", "quiet", "-print_format", "json")
	if err != nil {
		return nil, err
	}
	i := VideoInfo{}
	err = json.Unmarshal(res, &i)
	return i.Streams, err
}

func (v *Video) GetSubtitleStreams() ([]Stream, error) {
	// TODO: Use InfoStruct instead.
	res, err := shell.ExecuteCommand(v.l, "ffprobe", v.path, "-select_streams", "s", "-show_streams", "-v", "quiet", "-print_format", "json")
	if err != nil {
		return nil, err
	}
	i := VideoInfo{}
	err = json.Unmarshal(res, &i)
	return i.Streams, err
}
