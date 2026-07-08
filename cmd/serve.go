package cmd

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/0377/m3u8/api"
	_ "github.com/0377/m3u8/api/task"
)

func RunServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)

	port := fs.Int("port", 8080, "监听端口")
	dataDir := fs.String("data-dir", "./data", "任务存储目录")
	authEnabled := fs.Bool("auth-enabled", false, "启用 API Key 认证")
	apiKey := fs.String("api-key", "", "API Key")
	corsOrigins := fs.String("cors-origins", "*", "CORS 来源（逗号分隔）")
	maxTasks := fs.Int("max-tasks", 3, "最大并发任务数")
	taskTTL := fs.Duration("task-ttl", 24*time.Hour, "任务保留时长")
	cleanupInterval := fs.Duration("cleanup-interval", time.Hour, "过期清理间隔")
	showHelp := false
	fs.BoolVar(&showHelp, "h", false, "显示帮助信息")
	fs.BoolVar(&showHelp, "help", false, "显示帮助信息")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "M3U8 HTTP API 服务 v%s\n\n用法:\n  m3u8 serve [选项]\n\n选项:\n", Version)
		fs.PrintDefaults()
	}

	fs.Parse(args)

	if showHelp {
		fs.Usage()
		os.Exit(0)
	}

	if *authEnabled && *apiKey == "" {
		fmt.Fprintln(os.Stderr, "错误: --auth-enabled 时必须指定 --api-key")
		os.Exit(1)
	}

	origins := strings.Split(*corsOrigins, ",")
	srv, err := api.NewServer(api.ServerConfig{
		Port:            *port,
		DataDir:         *dataDir,
		AuthEnabled:     *authEnabled,
		APIKey:          *apiKey,
		CORSOrigins:     origins,
		MaxTasks:        *maxTasks,
		TaskTTL:         *taskTTL,
		CleanupInterval: *cleanupInterval,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("m3u8 API server listening on :%d", *port)
	log.Fatal(srv.ListenAndServe())
}
