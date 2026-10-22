package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KuronekoServer/RSS-BOT/internal/bot"
	"github.com/KuronekoServer/RSS-BOT/internal/config"
	"github.com/KuronekoServer/RSS-BOT/internal/store"
)

func main() {
	configPath := flag.String("config", "config.json", "設定ファイルのパス")
	flag.Parse()

	if err := run(*configPath); err != nil {
		slog.Error("終了します", "err", err)
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	st, err := store.New(connectCtx, cfg.Mongo.Link, cfg.Mongo.Database)
	cancel()
	if err != nil {
		return err
	}
	defer func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = st.Close(closeCtx)
	}()
	slog.Info("MongoDBに接続しました", "database", cfg.Mongo.Database)

	b, err := bot.New(cfg, st)
	if err != nil {
		return err
	}
	return b.Run(ctx)
}

