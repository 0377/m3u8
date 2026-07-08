package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/0377/m3u8/dl"
)

const version = "1.2.0"

var (
	url      string
	output   string
	chanSize int
	toMP4    bool
	showHelp bool
)

func init() {
	flag.StringVar(&url, "u", "", "M3U8 地址（必填）")
	flag.StringVar(&output, "o", "", "输出目录（必填）")
	flag.IntVar(&chanSize, "c", 25, "下载并发数")
	flag.BoolVar(&toMP4, "mp4", true, "合并后转 MP4（默认开启，使用 -mp4=false 关闭）")
	flag.BoolVar(&showHelp, "h", false, "显示帮助信息")
	flag.BoolVar(&showHelp, "help", false, "显示帮助信息")
	flag.Usage = usage
}

func main() {
	flag.Parse()

	if showHelp {
		usage()
		os.Exit(0)
	}

	if url == "" || output == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -u 和 -o 参数")
		fmt.Fprintln(os.Stderr)
		usage()
		os.Exit(1)
	}
	if chanSize <= 0 {
		fmt.Fprintln(os.Stderr, "错误: 参数 -c 必须大于 0")
		os.Exit(1)
	}

	downloader, err := dl.NewTask(output, url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if err := downloader.Start(chanSize, toMP4); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
}

func usage() {
	fmt.Fprintf(os.Stderr, `M3U8 下载工具 v%s - 下载并合并 TS 分片

用法:
  m3u8 -u <URL> -o <目录> [选项]

选项:
`, version)
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
示例:
  m3u8 -u=https://example.com/index.m3u8 -o=./output
  m3u8 -u https://example.com/index.m3u8 -o ./output -c 10

说明:
  - 仅支持 VOD 类型 M3U8
  - 合并后的文件保存为 <目录>/main.ts，默认同时输出 main.mp4
  - 转 MP4 需要系统已安装 ffmpeg
  - 部分链接限制请求频率，可适当调低 -c 并发数
`)
}
