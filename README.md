# dialog-extractor
Lightweight tool to extract dialog audio from videos

## Usage

1. Install golang
2. Install ffmpeg

```shell
go run extract_dialog.go ./path/to/video.mkv
```

Currently this only supports MKV videos with SRT subtitles embeded. This should work with most audio codecs but I have only tested AAC right now.
