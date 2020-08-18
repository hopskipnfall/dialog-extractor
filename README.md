# dialog-extractor
Lightweight tool to extract dialog audio from videos using subtitles for dialog timing reference.

## Usage

1. Install golang
2. Install ffmpeg

```shell
$ go run extract_dialog.go ./path/to/video.mkv
```

This currently works only with `.mkv` files. I have tested with SRT and ASS subtitles. It should work with most audio codecs but I have only tested AAC so far.

## Running tests

```shell
$ go test ./...
```

## TODO: Implement these features

- Support for extraction on all videos in a folder
- Identifying and stripping out opening/ending songs
- Add padding around subtitle timing in case the timing is not exact
- Add flags support
