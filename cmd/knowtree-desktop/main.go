// knowtree-desktop：知树桌面客户端（Wails v2 原生窗口 + 本地 HTTP 服务）。
//
// 复用服务端全部能力：启动时在 127.0.0.1 上挑一个空闲端口起真实 HTTP 服务，
// 再用 WebView2 原生窗口跳转加载。之所以不直接走 wails 的自定义协议：
// Windows 下其响应为整体缓冲，无法支撑 SSE 流式讲解；走本机回环 HTTP 则与浏览器行为完全一致。
// 关闭窗口即退出应用。数据目录默认在可执行文件同级的 data/ 下。
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/nabin-qq273274877/knowtree/internal/api"
	"github.com/nabin-qq273274877/knowtree/internal/config"
	"github.com/nabin-qq273274877/knowtree/internal/db"
)

// 构建期经 -ldflags 注入
var (
	version   = "dev"
	buildTime = "unknown"
	commit    = "unknown"
)

func main() {
	// 数据目录默认取 exe 同级 data/（双击/开始菜单启动时工作目录不可靠）
	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("[desktop] resolve exe: %v", err)
	}
	defaultData := filepath.Join(filepath.Dir(exePath), "data")

	dataDir := flag.String("data", defaultData, "数据目录")
	width := flag.Int("width", 1200, "窗口宽度")
	height := flag.Int("height", 780, "窗口高度")
	// 开发调试时可固定端口（配合 vite dev 的 /api 代理）；默认随机空闲端口
	addr := flag.String("addr", "", "固定 HTTP 监听地址（如 127.0.0.1:6010），留空则随机")
	flag.Parse()

	if err := os.MkdirAll(*dataDir, 0o755); err != nil {
		log.Fatalf("[desktop] create data dir %s: %v", *dataDir, err)
	}

	// 桌面版以 -H windowsgui 构建，没有控制台；日志写入数据目录便于排查
	if f, ferr := os.OpenFile(filepath.Join(*dataDir, "knowtree-desktop.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); ferr == nil {
		defer f.Close()
		log.SetOutput(io.MultiWriter(f, os.Stderr))
	}
	log.Printf("[desktop] v%s starting (data: %s)", version, *dataDir)

	sqlDB, err := db.Open(*dataDir)
	if err != nil {
		log.Fatalf("[desktop] init db: %v", err)
	}
	gdb, err := gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		log.Fatalf("[desktop] init gorm: %v", err)
	}

	cfg := &config.Config{Version: version, BuildTime: buildTime, Commit: commit}
	r := api.NewRouter(cfg, gdb, *dataDir)

	// 在本机回环地址上挑一个空闲端口并直接持有 listener，避免竞态
	listenAddr := *addr
	if listenAddr == "" {
		listenAddr = "127.0.0.1:0"
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("[desktop] listen %s: %v", listenAddr, err)
	}
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", ln.Addr().(*net.TCPAddr).Port)

	go func() {
		log.Printf("[desktop] serving on %s", baseURL)
		if err := r.RunListener(ln); err != nil && err != http.ErrServerClosed {
			log.Printf("[desktop] server exited: %v", err)
		}
	}()

	// 等待服务就绪再开窗口
	waitCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	for {
		req, _ := http.NewRequestWithContext(waitCtx, http.MethodGet, baseURL+"/api/health", nil)
		resp, derr := http.DefaultClient.Do(req)
		if derr == nil {
			resp.Body.Close()
			break
		}
		select {
		case <-waitCtx.Done():
			log.Fatalf("[desktop] server not ready: %v", derr)
		case <-time.After(100 * time.Millisecond):
		}
	}

	log.Println("[desktop] opening window")
	werr := wails.Run(&options.App{
		Title:     "知树·KnowTree",
		Width:     *width,
		Height:    *height,
		MinWidth:  880,
		MinHeight: 600,
		// 单实例锁：重复启动时聚焦已开窗口，避免两个进程写同一个 SQLite
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId: "cn.knowtree.desktop",
			OnSecondInstanceLaunch: func(_ options.SecondInstanceData) {
				log.Println("[desktop] second instance launch ignored")
			},
		},
		// 窗口首帧经中间件 302 到本地服务，之后所有请求（含 SSE）都走真实 HTTP 回环，
		// 与浏览器行为一致；Handler 作为兜底保留。
		AssetServer: &assetserver.Options{
			Handler: r,
			Middleware: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					target := baseURL + req.URL.RequestURI()
					if req.URL.RawQuery != "" {
						target = baseURL + req.URL.RequestURI()
					}
					http.Redirect(w, req, target, http.StatusTemporaryRedirect)
					_ = next // 不再回落到默认资产链路
				})
			},
		},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
			DisableWindowIcon:    false,
		},
		OnShutdown: func(_ context.Context) {
			log.Println("[desktop] window closed, shutting down")
			_ = sqlDB.Close()
		},
	})
	if werr != nil {
		log.Fatalf("[desktop] wails: %v", werr)
	}
}
