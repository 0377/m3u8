package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/0377/m3u8/dl"
)

const Version = "1.2.0"

func RunDownload(args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)

	var (
		url      string
		output   string
		filename string
		chanSize int
		maxRetry int
		toMP4    bool
		showHelp bool
	)

	fs.StringVar(&url, "u", "", "M3U8 地址（必填）")
	fs.StringVar(&output, "o", ".", "输出目录（默认当前目录）")
	fs.StringVar(&filename, "f", "main", "输出文件名（可带 .ts/.mp4 扩展名，默认 main）")
	fs.IntVar(&chanSize, "c", 25, "下载并发数")
	fs.IntVar(&maxRetry, "r", 10, "单分片最大重试次数")
	fs.BoolVar(&toMP4, "mp4", true, "合并后转 MP4（默认开启，使用 -mp4=false 关闭）")
	fs.BoolVar(&showHelp, "h", false, "显示帮助信息")
	fs.BoolVar(&showHelp, "help", false, "显示帮助信息")
	fs.Usage = func() { downloadUsage(fs) }

	fs.Parse(args)

	if showHelp {
		downloadUsage(fs)
		os.Exit(0)
	}

	if url == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -u 参数")
		fmt.Fprintln(os.Stderr)
		downloadUsage(fs)
		os.Exit(1)
	}
	if chanSize <= 0 {
		fmt.Fprintln(os.Stderr, "错误: 参数 -c 必须大于 0")
		os.Exit(1)
	}
	if maxRetry < 0 {
		fmt.Fprintln(os.Stderr, "错误: 参数 -r 不能小于 0")
		os.Exit(1)
	}

	downloader, err := dl.NewTask(output, url, filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	if err := downloader.Start(chanSize, toMP4, maxRetry); err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Done!")
}

func downloadUsage(fs *flag.FlagSet) {
	fmt.Fprintf(os.Stderr, `M3U8 下载工具 v%s - 下载并合并 TS 分片

用法:
  m3u8 -u <URL> [选项]

选项:
`, Version)
	fs.PrintDefaults()
	fmt.Fprintf(os.Stderr, `
示例:
  m3u8 -u=https://example.com/index.m3u8
  m3u8 -u=https://example.com/index.m3u8 -o=./output
  m3u8 -u https://example.com/index.m3u8 -o ./output -f myvideo

说明:
  - 仅支持 VOD 类型 M3U8
  - -f 指定输出文件名，合并为 <目录>/<名称>.ts，转 MP4 时为 <目录>/<名称>.mp4
  - 转 MP4 需要系统已安装 ffmpeg
  - 部分链接限制请求频率，可适当调低 -c 并发数或提高 -r 重试次数
`)
}
