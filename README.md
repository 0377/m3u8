# M3U8

M3U8 - a mini M3U8 downloader written in Golang for downloading and merging TS(Transport Stream) files.

You only need to specify the flags(`u`, `o`, `c`) to run, downloader will automatically download all TS files and consolidate them into a single TS file.

[中文说明](README_zh-CN.md)

## Features

- Download and parse M3U8（VOD）
- Retry on download TS failure
- Parse Master playlist
- Decrypt TS
- Merge TS

## Usage

```bash
make vendor && make build
./m3u8 -u=http://example.com/index.m3u8 -o=/data/example
```

Or run from source:

```bash
go run -mod=vendor . -u=http://example.com/index.m3u8 -o=/data/example
```

### binary

Linux & MacOS

```
./m3u8 -u=http://example.com/index.m3u8 -o=/data/example
```

Windows PowerShell

```
.\m3u8.exe -u="http://example.com/index.m3u8" -o="D:\data\example"
```

Flags:

```
-u  M3U8 URL (required)
-o  Output folder (required)
-c  Concurrency, default 25
```

## Development

Go Modules + vendor mode (`-mod=vendor`) for reproducible offline builds.

```bash
make vendor   # populate vendor/
make build    # build binary
make test     # run tests
```

Cross-compile:

```bash
make build-linux
make build-darwin-arm64
make build-windows
```

## Download

[Binary packages](https://github.com/0377/m3u8/releases)

[Upstream releases](https://github.com/oopsguy/m3u8/releases)
## Screenshots

![Demo](./screenshots/demo.gif)

## References

- [TS科普 2 包头](https://blog.csdn.net/cabbage2008/article/details/49281729)
- [HTTP Live Streaming draft-pantos-http-live-streaming-23](https://tools.ietf.org/html/draft-pantos-http-live-streaming-23#section-4.3.4.2)
- [MPEG transport stream - Wikipedia](https://en.wikipedia.org/wiki/MPEG_transport_stream)


## License

[MIT License](./LICENSE)