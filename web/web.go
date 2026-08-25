// Package web 内嵌前端构建产物（go:embed all:dist）。
// 发布构建前由脚本把 frontend/dist 拷贝到本目录；仓库内保留占位文件保证 go build 可用。
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// Dist 返回 dist 子目录的文件系统；不存在 index.html 时由调用方兜底。
func Dist() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
