package config

import (
	"flag"
	"os"
)

// Config 汇总运行配置：命令行 flag 优先，环境变量兜底。
type Config struct {
	Addr      string // 监听地址，默认 127.0.0.1:6006
	DataDir   string // 数据目录（SQLite + 上传文件），默认 ./data
	Version   string
	BuildTime string
	Commit    string
}

func Load(version, buildTime, commit string) *Config {
	addr := flag.String("addr", envOr("KNOWTREE_ADDR", "127.0.0.1:6006"), "HTTP 监听地址")
	data := flag.String("data", envOr("KNOWTREE_DATA_DIR", "./data"), "数据目录")
	flag.Parse()
	return &Config{
		Addr:      *addr,
		DataDir:   *data,
		Version:   version,
		BuildTime: buildTime,
		Commit:    commit,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
