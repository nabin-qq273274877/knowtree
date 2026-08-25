package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

func (s *Server) health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (s *Server) version(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name":       "knowtree",
		"version":    s.cfg.Version,
		"build_time": s.cfg.BuildTime,
		"commit":     s.cfg.Commit,
	})
}

func (s *Server) search(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" {
		c.JSON(http.StatusOK, []models.Node{})
		return
	}
	like := "%" + escapeLike(q) + "%"
	var nodes []models.Node
	if err := s.db.
		Where("title LIKE ? ESCAPE '\\'", like).
		Order("updated_at DESC").
		Limit(50).
		Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

// escapeLike 转义 LIKE 通配符，配合 ESCAPE '\' 使用。
func escapeLike(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return r.Replace(s)
}

// ---- settings（KV，value 为任意 JSON）----

func (s *Server) getSettings(c *gin.Context) {
	var rows []models.Setting
	if err := s.db.Find(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := map[string]any{}
	for _, r := range rows {
		var v any
		if err := json.Unmarshal([]byte(r.ValueJSON), &v); err != nil {
			v = r.ValueJSON
		}
		out[r.Key] = v
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) putSettings(c *gin.Context) {
	var body map[string]json.RawMessage
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now().Unix()
	for k, raw := range body {
		row := models.Setting{Key: k, ValueJSON: string(raw)}
		if err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value_json"}),
		}).Create(&row).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	_ = now

	// 返回合并后的完整配置
	s.getSettings(c)
}
