package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/dl"
	"github.com/0377/m3u8/tool"
)

const Version = "1.2.0"

type headerList []string

func (h *headerList) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerList) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func RunDownload(args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)

	var (
		url           string
		output        string
		filename      string
		chanSize      int
		maxRetry      int
		toMP4         bool
		showHelp      bool
		headerLines   headerList
		cookie        string
		insecureTLS   bool
		decryptScript string
		decryptConfig string
		scriptsDir    string
	)

	fs.StringVar(&url, "u", "", "M3U8 地址（必填）")
	fs.StringVar(&output, "o", ".", "输出目录（默认当前目录）")
	fs.StringVar(&filename, "f", "main", "输出文件名（可带 .ts/.mp4 扩展名，默认 main）")
	fs.IntVar(&chanSize, "c", 25, "下载并发数")
	fs.IntVar(&maxRetry, "r", 10, "单分片最大重试次数")
	fs.BoolVar(&toMP4, "mp4", true, "合并后转 MP4（默认开启，使用 -mp4=false 关闭）")
	fs.Var(&headerLines, "H", "自定义 HTTP 请求头，格式 \"Key: Value\"，可重复指定")
	fs.StringVar(&cookie, "cookie", "", "自定义 Cookie 请求头")
	fs.BoolVar(&insecureTLS, "k", false, "跳过 HTTPS 证书验证（不安全，仅用于自签名证书等场景）")
	fs.StringVar(&decryptScript, "decrypt-script", "", "解密脚本路径（.star 或 .py）")
	fs.StringVar(&decryptConfig, "decrypt-config", "decrypt.yaml", "解密配置文件路径")
	fs.StringVar(&scriptsDir, "scripts-dir", "scripts", "解密脚本库目录")
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

	httpCfg, err := buildHTTPConfig(headerLines, cookie, insecureTLS)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	cryptSvc, err := crypt.BuildService(crypt.ServiceOptions{
		DecryptScript: decryptScript,
		DecryptConfig: decryptConfig,
		ScriptsDir:    scriptsDir,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	downloader, err := dl.NewTask(output, url, filename, httpCfg, cryptSvc)
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

func buildHTTPConfig(headerLines headerList, cookie string, insecureTLS bool) (*tool.HTTPConfig, error) {
	headers, err := tool.ParseHeaders(headerLines)
	if err != nil {
		return nil, err
	}
	if len(headers) == 0 && cookie == "" && !insecureTLS {
		return nil, nil
	}
	return &tool.HTTPConfig{
		Headers:     headers,
		Cookie:      cookie,
		InsecureTLS: insecureTLS,
	}, nil
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
  m3u8 -u https://example.com/index.m3u8 -H "Referer: https://example.com/" -cookie "session=abc"
  m3u8 -u https://self-signed.example.com/index.m3u8 -k
  m3u8 -u https://example.com/index.m3u8 -decrypt-script scripts/custom.star
  m3u8 -u https://example.com/index.m3u8 -decrypt-config decrypt.yaml -scripts-dir scripts

说明:
  - 仅支持 VOD 类型 M3U8
  - 中断后使用相同 -u、-o、-f 重新运行可自动续传
  - -f 指定输出文件名，合并为 <目录>/<名称>.ts，转 MP4 时为 <目录>/<名称>.mp4
  - 转 MP4 需要系统已安装 ffmpeg
  - 部分链接限制请求频率，可适当调低 -c 并发数或提高 -r 重试次数
  - -H 可多次指定自定义请求头；-cookie 设置 Cookie；-k 跳过 HTTPS 证书验证
  - -decrypt-script 指定解密脚本；-decrypt-config 指定解密配置（默认 decrypt.yaml）
  - -scripts-dir 指定脚本库目录（默认 scripts），按 METHOD/域名自动匹配
`)
}
