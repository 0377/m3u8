package main

import (
	"fmt"
	"os"

	"github.com/0377/m3u8/cmd"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "serve":
			cmd.RunServe(os.Args[2:])
			return
		case "help", "-h", "--help":
			printUsage()
			return
		}
	}
	cmd.RunDownload(os.Args[1:])
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `M3U8 工具 v%s

用法:
  m3u8 -u <URL> [选项]          下载 M3U8（CLI 模式）
  m3u8 serve [选项]             启动 HTTP API 服务

运行 m3u8 serve -h 查看服务选项
`, cmd.Version)
}
