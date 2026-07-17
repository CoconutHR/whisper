package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"whisper/internal/chat"
	webapp "whisper/internal/web"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "监听地址")
	dataPath := flag.String("data", "data/state.json", "数据文件")
	staticDir := flag.String("static", ".", "前端静态文件目录")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	absoluteDataPath, err := filepath.Abs(*dataPath)
	if err != nil {
		logger.Error("解析数据路径失败", "error", err)
		os.Exit(1)
	}
	store, err := chat.NewStore(absoluteDataPath)
	if err != nil {
		logger.Error("初始化数据存储失败", "error", err)
		os.Exit(1)
	}

	server := webapp.NewServer(webapp.Config{Address: *address, StaticDir: *staticDir}, store, logger)
	logger.Info("whisper（耳语）已启动", "address", "http://"+*address, "data", absoluteDataPath)
	if err := http.ListenAndServe(*address, server.Handler()); err != nil {
		logger.Error("服务器停止", "error", err)
		os.Exit(1)
	}
}
