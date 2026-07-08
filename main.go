package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/dl"
	"github.com/0377/m3u8/tool"
)

const version = "1.2.0"

var (
	url         string
	output      string
	filename    string
	chanSize    int
	maxRetry    int
	toMP4       bool
	showHelp    bool
	headerLines headerList
	cookie        string
	insecureTLS   bool
	decryptScript string
	decryptConfig string
	scriptsDir    string
)

type headerList []string

func (h *headerList) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerList) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func init() {
	flag.StringVar(&url, "u", "", "M3U8 地址（必填）")
	flag.StringVar(&output, "o", ".", "输出目录（默认当前目录）")
	flag.StringVar(&filename, "f", "main", "输出文件名（可带 .ts/.mp4 扩展名，默认 main）")
	flag.IntVar(&chanSize, "c", 25, "下载并发数")
	flag.IntVar(&maxRetry, "r", 10, "单分片最大重试次数")
	flag.BoolVar(&toMP4, "mp4", true, "合并后转 MP4（默认开启，使用 -mp4=false 关闭）")
	flag.Var(&headerLines, "H", "自定义 HTTP 请求头，格式 \"Key: Value\"，可重复指定")
	flag.StringVar(&cookie, "cookie", "", "自定义 Cookie 请求头")
	flag.BoolVar(&insecureTLS, "k", false, "跳过 HTTPS 证书验证（不安全，仅用于自签名证书等场景）")
	flag.StringVar(&decryptScript, "decrypt-script", "", "解密脚本路径（.star 或 .py）")
	flag.StringVar(&decryptConfig, "decrypt-config", "decrypt.yaml", "解密配置文件路径")
	flag.StringVar(&scriptsDir, "scripts-dir", "scripts", "解密脚本库目录")
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

	if url == "" {
		fmt.Fprintln(os.Stderr, "错误: 必须指定 -u 参数")
		fmt.Fprintln(os.Stderr)
		usage()
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

	httpCfg, err := buildHTTPConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		os.Exit(1)
	}

	cryptSvc, err := buildCryptService()
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

func buildCryptService() (*crypt.Service, error) {
	cfg, err := crypt.LoadConfig(decryptConfig)
	if err != nil {
		return nil, err
	}
	scriptsDirVal := scriptsDir
	if cfg != nil && cfg.ScriptsDir != "" {
		scriptsDirVal = cfg.ScriptsDir
	}
	scriptsAbs, _ := filepath.Abs(scriptsDirVal)
	timeout := 30 * time.Second
	if cfg != nil && cfg.ExternalTimeout > 0 {
		timeout = cfg.ExternalTimeout
	}
	reg, err := crypt.NewRegistry(crypt.RegistryOptions{
		ScriptsDir:      scriptsDirVal,
		ScriptsDirAbs:   scriptsAbs,
		CLIScript:       decryptScript,
		Config:          cfg,
		ExternalTimeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return crypt.NewService(reg), nil
}

func buildHTTPConfig() (*tool.HTTPConfig, error) {
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

func usage() {
	fmt.Fprintf(os.Stderr, `M3U8 下载工具 v%s - 下载并合并 TS 分片

用法:
  m3u8 -u <URL> [选项]

选项:
`, version)
	flag.PrintDefaults()
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
  - -f 指定输出文件名，合并为 <目录>/<名称>.ts，转 MP4 时为 <目录>/<名称>.mp4
  - 转 MP4 需要系统已安装 ffmpeg
  - 部分链接限制请求频率，可适当调低 -c 并发数或提高 -r 重试次数
  - -H 可多次指定自定义请求头；-cookie 设置 Cookie；-k 跳过 HTTPS 证书验证
  - -decrypt-script 指定解密脚本；-decrypt-config 指定解密配置（默认 decrypt.yaml）
  - -scripts-dir 指定脚本库目录（默认 scripts），按 METHOD/域名自动匹配
`)
}
