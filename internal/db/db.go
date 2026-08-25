package db

import (
	"database/sql"
	"embed"
	"fmt"
	"log"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "github.com/glebarez/go-sqlite" // 纯 Go SQLite 驱动（modernc 分支），注册名 "sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Open 打开 SQLite（WAL + busy_timeout + FK），并执行 goose 迁移。
// 单用户应用：连接池限制为 1，写天然串行化，从根上规避锁竞争。
func Open(dataDir string) (*sql.DB, error) {
	abs, err := filepath.Abs(filepath.Join(dataDir, "knowtree.db"))
	if err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.ToSlash(abs) +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_pragma=foreign_keys(ON)"

	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)
	if err := d.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if err := migrate(d); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	log.Printf("[db] ready: %s (wal)", abs)
	return d, nil
}

func migrate(d *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}
	return goose.Up(d, "migrations")
}
