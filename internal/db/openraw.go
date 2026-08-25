package db

import (
	"database/sql"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

// OpenRaw 以独立连接打开一个 SQLite 文件（用于恢复前的备份文件校验等场景）。
func OpenRaw(path string) (*sql.DB, error) {
	dsn := "file:" + path + "?_pragma=busy_timeout(3000)"
	d, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if err := d.Ping(); err != nil {
		d.Close()
		return nil, fmt.Errorf("无法打开数据库：%w", err)
	}
	return d, nil
}
