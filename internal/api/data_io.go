package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/nabin-qq273274877/knowtree/internal/db"
	"github.com/nabin-qq273274877/knowtree/internal/models"
)

// openSQLiteFile 以独立连接打开一个 SQLite 文件（恢复前的备份校验用）。
func openSQLiteFile(path string) (*gorm.DB, error) {
	sqlDB, err := db.OpenRaw(path)
	if err != nil {
		return nil, err
	}
	return gorm.Open(sqlite.Dialector{Conn: sqlDB}, &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
}

// ---- GET /api/export （全量导出 JSON，FR-8）----

type exportPayload struct {
	App         string              `json:"app"`
	Format      int                 `json:"format"`
	ExportedAt  int64               `json:"exported_at"`
	Version     string              `json:"version,omitempty"`
	Nodes       []models.Node       `json:"nodes"`
	Edges       []models.Edge       `json:"edges"`
	Resources   []models.Resource   `json:"resources"`
	Exercises   []models.Exercise   `json:"exercises"`
	Annotations []models.Annotation `json:"annotations"`
}

func (s *Server) exportAll(c *gin.Context) {
	payload := exportPayload{
		App:        "knowtree",
		Format:     1,
		ExportedAt: time.Now().Unix(),
		Version:    s.cfg.Version,
	}
	s.db.Order("created_at ASC").Find(&payload.Nodes)
	s.db.Find(&payload.Edges)
	s.db.Find(&payload.Resources)
	s.db.Find(&payload.Exercises)
	s.db.Find(&payload.Annotations)

	filename := "knowtree-export-" + time.Now().Format("20060102-150405") + ".json"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.JSON(http.StatusOK, payload)
}

// ---- POST /api/import （整体替换导入；不含 settings，保留本地 LLM 配置）----

func (s *Server) importAll(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少上传文件字段 file"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer f.Close()

	var payload exportPayload
	if err := json.NewDecoder(f).Decode(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "JSON 解析失败：" + err.Error()})
		return
	}
	if payload.App != "knowtree" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是 knowtree 的导出文件"})
		return
	}

	s.replaceWithData(&payload.Nodes, &payload.Edges, &payload.Resources, &payload.Exercises, &payload.Annotations, c)

	c.JSON(http.StatusOK, gin.H{
		"restored": true,
		"counts": gin.H{
			"nodes": len(payload.Nodes), "edges": len(payload.Edges),
			"resources": len(payload.Resources), "exercises": len(payload.Exercises),
			"annotations": len(payload.Annotations),
		},
	})
}

// replaceWithData 在事务中清空并重写全部业务表（导入/恢复共用）。
func (s *Server) replaceWithData(nodes *[]models.Node, edges *[]models.Edge, resources *[]models.Resource, exercises *[]models.Exercise, annotations *[]models.Annotation, c *gin.Context) bool {
	s.db.Exec("PRAGMA foreign_keys = OFF")
	tx := s.db.Begin()
	defer func() {
		tx.Rollback()
		s.db.Exec("PRAGMA foreign_keys = ON")
	}()
	for _, stmt := range []string{
		"DELETE FROM annotations", "DELETE FROM exercises", "DELETE FROM resources",
		"DELETE FROM edges", "DELETE FROM nodes",
	} {
		if err := tx.Exec(stmt).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return false
		}
	}
	for _, n := range *nodes {
		if err := tx.Create(&n).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "节点写入失败：" + err.Error()})
			return false
		}
	}
	for i := range *edges {
		if err := tx.Create(&(*edges)[i]).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "连线写入失败：" + err.Error()})
			return false
		}
	}
	for i := range *resources {
		if err := tx.Create(&(*resources)[i]).Error; err != nil {
			continue // 非致命：跳过悬空引用等
		}
	}
	for i := range *exercises {
		if err := tx.Create(&(*exercises)[i]).Error; err != nil {
			continue
		}
	}
	for i := range *annotations {
		if err := tx.Create(&(*annotations)[i]).Error; err != nil {
			continue
		}
	}
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return false
	}
	s.db.Exec("PRAGMA foreign_keys = ON")
	return true
}

// ---- GET /api/backup （SQLite 一致性快照下载，S5 / §3.6）----

func (s *Server) backupDB(c *gin.Context) {
	tmp := filepath.Join(s.dataDir, ".backup.tmp")
	os.Remove(tmp)
	if err := s.db.Exec("VACUUM INTO ?", tmp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "备份失败：" + err.Error()})
		return
	}
	filename := "knowtree-backup-" + time.Now().Format("20060102-150405") + ".db"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	defer os.Remove(tmp)
	c.File(tmp)
}

// ---- POST /api/restore （从上传的 .db 快照在线恢复）----

var restoreTables = []string{"settings", "annotations", "exercises", "resources", "edges", "nodes"}

func (s *Server) restoreDB(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少上传文件字段 file"})
		return
	}
	staging := filepath.Join(s.dataDir, ".restore-staging.db")
	if err := c.SaveUploadedFile(fileHeader, staging); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer os.Remove(staging)

	check, err := openSQLiteFile(staging)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "不是有效的 SQLite 文件：" + err.Error()})
		return
	}
	var rNodes []models.Node
	var rEdges []models.Edge
	var rRes []models.Resource
	var rEx []models.Exercise
	var rAnn []models.Annotation
	ok := true
	for _, t := range restoreTables {
		var cnt int64
		check.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?", t).Scan(&cnt)
		if cnt == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "文件缺少表 " + t + "，不是 knowtree 备份"})
			ok = false
			break
		}
	}
	if ok {
		check.Order("created_at ASC").Find(&rNodes)
		check.Find(&rEdges)
		check.Find(&rRes)
		check.Find(&rEx)
		check.Find(&rAnn)
	}
	if sqlDB, err2 := check.DB(); err2 == nil {
		sqlDB.Close()
	}
	if !ok {
		return
	}

	if !s.replaceWithData(&rNodes, &rEdges, &rRes, &rEx, &rAnn, c) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"restored": true})
}

func escapeSingleQuotes(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\'' {
			out = append(out, '\'', '\'')
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}

// ---- GET /api/stats （统计概览 S8）----

func (s *Server) stats(c *gin.Context) {
	type StatusRow struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	type StageRow struct {
		Stage    string `json:"stage"`
		Total    int    `json:"total"`
		Mastered int    `json:"mastered"`
		Learning int    `json:"learning"`
	}

	var byStatus []StatusRow
	s.db.Raw(`SELECT status, COUNT(*) AS count FROM nodes GROUP BY status`).Scan(&byStatus)

	var byStage []StageRow
	s.db.Raw(`
		SELECT COALESCE(NULLIF(stage,''),'未设置') AS stage,
		       COUNT(*) AS total,
		       SUM(CASE WHEN status='mastered' THEN 1 ELSE 0 END) AS mastered,
		       SUM(CASE WHEN status IN ('learning','partial') THEN 1 ELSE 0 END) AS learning
		FROM nodes GROUP BY stage ORDER BY total DESC`).Scan(&byStage)

	var total, edgeCount, annCount, exCount int64
	s.db.Model(&models.Node{}).Count(&total)
	s.db.Model(&models.Edge{}).Count(&edgeCount)
	s.db.Model(&models.Annotation{}).Count(&annCount)
	s.db.Model(&models.Exercise{}).Count(&exCount)

	statusMap := map[string]int{}
	for _, r := range byStatus {
		statusMap[r.Status] = r.Count
	}
	mastered := statusMap["mastered"]
	pct := 0
	if total > 0 {
		pct = mastered * 100 / int(total)
	}
	c.JSON(http.StatusOK, gin.H{
		"total_nodes":     total,
		"mastered_pct":    pct,
		"by_status":       statusMap,
		"by_stage":        byStage,
		"edge_count":      edgeCount,
		"annotation_count": annCount,
		"exercise_count":   exCount,
	})
}
