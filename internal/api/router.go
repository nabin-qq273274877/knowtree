package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/nabin-qq273274877/knowtree/internal/config"
	web "github.com/nabin-qq273274877/knowtree/web"
)

type Server struct {
	cfg *config.Config
	db  *gorm.DB
}

func NewRouter(cfg *config.Config, gdb *gorm.DB) *gin.Engine {
	s := &Server{cfg: cfg, db: gdb}

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

		// 节点（知识点树）
		g.GET("/nodes", s.listNodes)
		g.POST("/nodes", s.createNode)
		g.PATCH("/nodes/:id", s.updateNode)
		g.DELETE("/nodes/:id", s.deleteNode)
		g.POST("/nodes/:id/move", s.moveNode)

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

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
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
