package api

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/nabin-qq273274877/knowtree/internal/config"
	web "github.com/nabin-qq273274877/knowtree/web"
)

// 前端未构建时的兜底页面（正常发布二进制内嵌的是真实前端，不会走到这里）
const fallbackHTML = `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="UTF-8"><title>knowtree</title></head>
<body style="font-family:system-ui;padding:40px;line-height:1.8">
<h2>knowtree 前端尚未构建</h2>
<p>当前二进制内含占位页面。请运行 <code>scripts/build.ps1</code>（或 build.sh）完成前端构建与编译。</p>
<p>开发模式请使用 Vite Dev Server（http://localhost:6006），API 已代理到本服务。REST API 可用：<code>GET /api/health</code></p>
</body></html>`

type Server struct {
	cfg     *config.Config
	db      *gorm.DB
	dataDir string
}

func NewRouter(cfg *config.Config, gdb *gorm.DB, dataDir string) *gin.Engine {
	s := &Server{cfg: cfg, db: gdb, dataDir: dataDir}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	g := r.Group("/api")
	{
		g.GET("/health", s.health)
		g.GET("/version", s.version)
		g.GET("/search", s.search)
		g.GET("/settings", s.getSettings)
		g.PUT("/settings", s.putSettings)

		g.POST("/nodes/positions", s.setPositions)

		// 节点（知识点树）
		g.GET("/nodes", s.listNodes)
		g.POST("/nodes", s.createNode)
		g.PATCH("/nodes/:id", s.updateNode)
		g.DELETE("/nodes/:id", s.deleteNode)
		g.POST("/nodes/:id/move", s.moveNode)

		// 教学资源（FR-5）
		g.GET("/nodes/:id/resources", s.listResources)
		g.POST("/nodes/:id/resources", s.createResource)
		g.DELETE("/resources/:id", s.deleteResource)

		// 批注 / 学习心得（FR-10）
		g.GET("/nodes/:id/annotations", s.listAnnotations)
		g.POST("/nodes/:id/annotations", s.createAnnotation)
		g.PATCH("/annotations/:id", s.updateAnnotation)
		g.DELETE("/annotations/:id", s.deleteAnnotation)

		// 练习题（FR-9）
		g.GET("/nodes/:id/exercises", s.listExercises)
		g.POST("/nodes/:id/exercises", s.createExercises)
		g.PATCH("/exercises/:id", s.updateExercise)
		g.DELETE("/exercises/:id", s.deleteExercise)
		g.POST("/exercises/:id/submit", s.submitExercise)

		// LLM（FR-6）
		g.POST("/llm/test", s.llmTest)
		g.POST("/llm/explain", s.llmExplain)
		g.POST("/llm/generate-subtree", s.llmGenerateSubtree)
		g.POST("/llm/generate-exercises", s.llmGenerateExercises)

		// 数据导入导出 / 备份恢复 / 统计（M5）
		g.GET("/export", s.exportAll)
		g.POST("/import", s.importAll)
		g.GET("/backup", s.backupDB)
		g.POST("/restore", s.restoreDB)
		g.GET("/stats", s.stats)

		// 版本与自更新（FR-11）
		g.POST("/update/check", s.updateCheck)
		g.POST("/update/apply", s.updateApply)
		g.POST("/update/restart", s.updateRestart)

		// 连线（自由关联，层级不在此表）
		g.GET("/edges", s.listEdges)
		g.POST("/edges", s.createEdge)
		g.PATCH("/edges/:id", s.updateEdge)
		g.DELETE("/edges/:id", s.deleteEdge)
	}

	registerSPA(r)
	return r
}

// registerSPA 托管内嵌前端产物：静态文件存在则直接返回，否则回退 index.html（前端路由）。
func registerSPA(r *gin.Engine) {
	dist, err := web.Dist()
	if err != nil {
		r.NoRoute(func(c *gin.Context) {
			c.String(http.StatusInternalServerError, "embed fs error: %v", err)
		})
		return
	}
	fileServer := http.FileServer(http.FS(dist))
	hasIndex := false
	if st, err := fs.Stat(dist, "index.html"); err == nil && !st.IsDir() {
		hasIndex = true
	}

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}

		if !hasIndex {
			c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(fallbackHTML))
			return
		}

		name := strings.TrimPrefix(p, "/")
		if name == "" {
			name = "index.html"
		}
		if f, err := dist.Open(name); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(c.Writer, c.Request)
			return
		}
		// SPA 回退到 index.html
		req := c.Request.Clone(c.Request.Context())
		req.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, req)
	})
}
