package main

import (
	"log"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/nabin-qq273274877/knowtree/internal/api"
	"github.com/nabin-qq273274877/knowtree/internal/config"
	"github.com/nabin-qq273274877/knowtree/internal/db"
)

// 构建期经 -ldflags "-X main.version=..." 注入。
var (
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"
)

func main() {
	cfg := config.Load(version, buildTime, commit)

	sqlDB, err := db.Open(cfg.DataDir)
	if err != nil {
		log.Fatalf("[knowtree] init db: %v", err)
	}

	gdb, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("[knowtree] init gorm: %v", err)
	}

	r := api.NewRouter(cfg, gdb)
	log.Printf("[knowtree] v%s listening on http://%s (data: %s)", cfg.Version, cfg.Addr, cfg.DataDir)
	if err := r.Run(cfg.Addr); err != nil {
		log.Fatal(err)
	}
}
