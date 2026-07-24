package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"

	"whisper/internal/blob"
	"whisper/internal/chat"
	webapp "whisper/internal/web"
)

func main() {
	address := flag.String("addr", "127.0.0.1:8080", "监听地址")
	databasePath := flag.String("db", "data/whisper.db", "SQLite 数据库文件")
	userBackupPath := flag.String("users-backup", "data/users-backup.json", "明文用户备份文件")
	staticDir := flag.String("static", ".", "前端静态文件目录")
	vapidPublicKey := flag.String("vapid-public-key", os.Getenv("WHISPER_VAPID_PUBLIC_KEY"), "Web Push VAPID 公钥")
	vapidPrivateKey := flag.String("vapid-private-key", os.Getenv("WHISPER_VAPID_PRIVATE_KEY"), "Web Push VAPID 私钥")
	vapidSubject := flag.String("vapid-subject", os.Getenv("WHISPER_VAPID_SUBJECT"), "Web Push 联系地址，例如 admin@example.com")
	generateVAPIDKeys := flag.Bool("generate-vapid-keys", false, "生成一组 VAPID 密钥后退出")
	flag.Parse()
	if *generateVAPIDKeys {
		privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
		if err != nil {
			fmt.Fprintln(os.Stderr, "生成 VAPID 密钥失败:", err)
			os.Exit(1)
		}
		fmt.Printf("WHISPER_VAPID_PUBLIC_KEY=%s\nWHISPER_VAPID_PRIVATE_KEY=%s\n", publicKey, privateKey)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	pushValues := []string{*vapidPublicKey, *vapidPrivateKey, *vapidSubject}
	pushConfigured := pushValues[0] != "" || pushValues[1] != "" || pushValues[2] != ""
	if pushConfigured && (pushValues[0] == "" || pushValues[1] == "" || pushValues[2] == "") {
		logger.Error("Web Push 配置不完整，需要同时设置公钥、私钥和联系地址")
		os.Exit(1)
	}
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
		VAPIDPublicKey: *vapidPublicKey, VAPIDPrivateKey: *vapidPrivateKey, VAPIDSubject: *vapidSubject,
	}, store, logger)
	logger.Info("whisper（耳语）已启动", "address", "http://"+*address,
		"database", store.DatabasePath(), "users_backup", store.UserBackupPath(),
		"r2_enabled", r2Enabled, "web_push_enabled", pushConfigured)
	if err := http.ListenAndServe(*address, server.Handler()); err != nil {
		logger.Error("服务器停止", "error", err)
		os.Exit(1)
	}
}
