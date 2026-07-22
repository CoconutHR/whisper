package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"

	"whisper/internal/blob"
	"whisper/internal/chat"
	webapp "whisper/internal/web"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "监听地址")
	databasePath := flag.String("db", "data/whisper.db", "SQLite 数据库文件")
	userBackupPath := flag.String("users-backup", "data/users-backup.json", "明文用户备份文件")
	staticDir := flag.String("static", ".", "前端静态文件目录")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store, err := chat.NewStore(chat.StoreConfig{
		DatabasePath: *databasePath, UserBackupPath: *userBackupPath,
	})
	if err != nil {
		logger.Error("初始化数据存储失败", "error", err)
		os.Exit(1)
	}
	defer store.Close()
	r2Config, r2Enabled, err := blob.R2ConfigFromEnv()
	if err != nil {
		logger.Error("读取 R2 配置失败", "error", err)
		os.Exit(1)
	}
	var objectStore blob.Store
	if r2Enabled {
		objectStore, err = blob.NewR2Store(context.Background(), r2Config)
		if err != nil {
			logger.Error("初始化 R2 失败", "error", err)
			os.Exit(1)
		}
	}

	server := webapp.NewServer(webapp.Config{
		Address: *address, StaticDir: *staticDir, ObjectStore: objectStore,
	}, store, logger)
	logger.Info("whisper（耳语）已启动", "address", "http://"+*address,
		"database", store.DatabasePath(), "users_backup", store.UserBackupPath(), "r2_enabled", r2Enabled)
	if err := http.ListenAndServe(*address, server.Handler()); err != nil {
		logger.Error("服务器停止", "error", err)
		os.Exit(1)
	}
}
