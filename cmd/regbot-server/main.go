// Command regbot-server starts the RegBot HTTP API server.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/api"
	"github.com/ErrLogic/regbot/internal/api/handler"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/device"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	migrateCmd := flag.String("migrate", "", "migration command: up or down")
	migrationsDir := flag.String("migrations", "migrations", "path to migrations directory")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	if *migrateCmd != "" {
		dsn := cfg.DB.DSN()
		if *migrateCmd == "up" {
			if err := db.RunMigrations(dsn, *migrationsDir); err != nil {
				log.Fatalf("migrate up: %v", err)
			}
			log.Println("migrations applied successfully")
		}
		return
	}

	if cfg.Server.JWTSecret == "" {
		log.Fatal("server.jwt_secret is required. Set via config or REGBOT_SERVER_JWT_SECRET env var.")
	}

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DB.DSN())
	if err != nil {
		log.Fatalf("connect to postgres: %v", err)
	}
	defer pool.Close()

	rdb, err := db.ConnectRedis(ctx, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Fatalf("connect to redis: %v", err)
	}
	defer func() { _ = rdb.Close() }()

	userStore := db.NewUserStore(pool)
	deviceStore := db.NewDeviceStore(pool)
	accountStore := db.NewAccountStore(pool)
	jobStore := db.NewJobStore(pool)
	jobLogStore := db.NewJobLogStore(pool)
	mediaStore := db.NewMediaStore(pool)

	adbClient := adb.New()
	deviceMgr := device.NewManager(adbClient, deviceStore)

	authH := handler.NewAuthHandler(userStore, cfg.Server.JWTSecret)
	deviceH := handler.NewDeviceHandler(deviceMgr)
	jobH := handler.NewJobHandler(jobStore, jobLogStore, accountStore, rdb)
	accountH := handler.NewAccountHandler(accountStore)
	mediaH := handler.NewMediaHandler(mediaStore)

	router := api.NewRouter(cfg.Server.JWTSecret, authH, deviceH, jobH, accountH, mediaH)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("RegBot server listening on %s", addr)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Println("server stopped")
}
