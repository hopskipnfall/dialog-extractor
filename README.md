# dialog-extractor
Lightweight tool to extract dialog audio from videos using subtitles for dialog timing reference.

## Usage

1. Install golang
2. Install ffmpeg

```shell
$ go run extract_dialog.go ./path/to/video.mkv
```

Currently this only supports MKV videos with SRT subtitles embeded. This should work with most audio codecs but I have only tested AAC right now.

## Running tests

```shell
$ go test ./...
```

## TODO: Implement these features

- Support for ASS subtitles (currently only SRT is supported)
- Interactive audio/subtitle track selection (first track is always selected)
- Support for extraction on all videos in a folder
- Identifying and stripping out opening/ending songs
- Add padding around subtitle timing in case the timing is not exact
