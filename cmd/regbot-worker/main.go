// Command regbot-worker runs the background job worker for RegBot.
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"github.com/ErrLogic/regbot/internal/adb"
	"github.com/ErrLogic/regbot/internal/automation"
	"github.com/ErrLogic/regbot/internal/config"
	"github.com/ErrLogic/regbot/internal/db"
	"github.com/ErrLogic/regbot/internal/device"
	"github.com/ErrLogic/regbot/internal/job"
	"github.com/ErrLogic/regbot/internal/session"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config file")
	poolSize := flag.Int("workers", 2, "number of concurrent workers")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
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

	jobStore := db.NewJobStore(pool)
	jobLogStore := db.NewJobLogStore(pool)
	deviceStore := db.NewDeviceStore(pool)

	adbClient := adb.New()
	deviceMgr := device.NewManager(adbClient, deviceStore)
	sessionPool := session.NewPool(cfg.Appium)
	autoSvc := automation.NewService(cfg, sessionPool)

	_ = deviceMgr
	_ = jobLogStore

	worker := job.NewWorker(rdb, jobStore, jobLogStore, autoSvc, *poolSize)

	sigCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	log.Printf("RegBot worker starting with %d workers", *poolSize)

	if _, err := deviceMgr.ListDevices(ctx); err != nil {
		log.Printf("warning: device scan failed: %v", err)
	}

	if err := worker.Run(sigCtx); err != nil {
		log.Printf("worker stopped: %v", err)
	}

	sessionPool.ReleaseAll(ctx)
	log.Println("worker stopped")
}
