package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

var validStatuses = map[string]bool{
	"not_started": true,
	"learning":    true,
	"partial":     true,
	"mastered":    true,
	"forgotten":   true,
}

// ---- DTO（集中定义，字段与前端 TS 类型对应）----

type createNodeReq struct {
	Title    string  `json:"title"`
	ParentID *string `json:"parent_id"`
	Stage    *string `json:"stage"`
}

type updateNodeReq struct {
	Title     *string  `json:"title"`
	ContentMd *string  `json:"content_md"`
	Status    *string  `json:"status"`
	Stage     *string  `json:"stage"`
	PosX      *float64 `json:"pos_x"`
	PosY      *float64 `json:"pos_y"`
}

type moveNodeReq struct {
	ParentID  *string  `json:"parent_id"` // null → 移到根
	SortOrder *float64 `json:"sort_order"`
}

type positionItem struct {
	ID   string  `json:"id"`
	PosX float64 `json:"pos_x"`
	PosY float64 `json:"pos_y"`
}

type setPositionsReq struct {
	Nodes []positionItem `json:"nodes"`
}

// ---- handlers ----

func (s *Server) listNodes(c *gin.Context) {
	var nodes []models.Node
	if err := s.db.Order("sort_order ASC, created_at ASC").Find(&nodes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nodes)
}

func (s *Server) createNode(c *gin.Context) {
	var req createNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if req.ParentID != nil && !s.nodeExists(req.ParentID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "parent_id not found"})
		return
	}
	now := time.Now().Unix()
	n := models.Node{
		ID:        uuid.NewString(),
		Title:     req.Title,
		Status:    "not_started",
		ParentID:  req.ParentID,
		Stage:     req.Stage,
		SortOrder: s.nextSortOrder(req.ParentID),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Create(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, n)
}

func (s *Server) updateNode(c *gin.Context) {
	id := c.Param("id")
	var n models.Node
	if err := s.db.First(&n, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var req updateNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statusChanged := false
	if req.Title != nil {
		if *req.Title == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "title cannot be empty"})
			return
		}
		n.Title = *req.Title
	}
	if req.ContentMd != nil {
		n.ContentMd = *req.ContentMd
	}
	if req.Status != nil {
		if !validStatuses[*req.Status] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
			return
		}
		if n.Status != *req.Status {
			n.Status = *req.Status
			now := time.Now().Unix()
			n.StatusChangedAt = &now
			statusChanged = true
		}
	}
	if req.Stage != nil {
		n.Stage = req.Stage
	}
	if req.PosX != nil {
		n.PosX = req.PosX
	}
	if req.PosY != nil {
		n.PosY = req.PosY
	}

	n.UpdatedAt = time.Now().Unix()
	if err := s.db.Save(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = statusChanged
	c.JSON(http.StatusOK, n)
}

func (s *Server) deleteNode(c *gin.Context) {
	id := c.Param("id")
	if !s.nodeExists(&id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}

	// 统计将级联删除的子树规模（含自身），供前端二次确认与结果提示
	var count int64
	s.db.WithContext(c).Raw(`
		WITH RECURSIVE sub(id) AS (
			SELECT id FROM nodes WHERE id = ?
			UNION ALL
			SELECT n.id FROM nodes n JOIN sub s ON n.parent_id = s.id
		) SELECT COUNT(*) FROM sub`, id).Scan(&count)

	if err := s.db.Delete(&models.Node{}, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": count})
}

func (s *Server) moveNode(c *gin.Context) {
	id := c.Param("id")
	var n models.Node
	if err := s.db.First(&n, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var req moveNodeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.ParentID != nil {
		if *req.ParentID == id {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot move node under itself"})
			return
		}
		if !s.nodeExists(req.ParentID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "target parent not found"})
			return
		}
		// 环检测：新父级的祖先链中不能包含自身
		if s.isAncestor(id, *req.ParentID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot move node under its own descendant"})
			return
		}
	}
	n.ParentID = req.ParentID
	if req.SortOrder != nil {
		n.SortOrder = *req.SortOrder
	} else {
		n.SortOrder = s.nextSortOrder(req.ParentID)
	}
	n.UpdatedAt = time.Now().Unix()
	if err := s.db.Save(&n).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, n)
}

// ---- helpers ----

// setPositions 批量保存画布坐标（拖拽落点、自动排布）。
func (s *Server) setPositions(c *gin.Context) {
	var req setPositionsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now().Unix()
	updated := 0
	for _, it := range req.Nodes {
		res := s.db.Model(&models.Node{}).Where("id = ?", it.ID).Updates(map[string]any{
			"pos_x":      it.PosX,
			"pos_y":      it.PosY,
			"updated_at": now,
		})
		if res.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
			return
		}
		updated += int(res.RowsAffected)
	}
	c.JSON(http.StatusOK, gin.H{"updated": updated})
}

func (s *Server) nodeExists(id *string) bool {
	var cnt int64
	s.db.Model(&models.Node{}).Where("id = ?", *id).Count(&cnt)
	return cnt > 0
}

func (s *Server) nextSortOrder(parentID *string) float64 {
	var max sql.NullFloat64
	q := s.db.Model(&models.Node{})
	if parentID != nil {
		q = q.Where("parent_id = ?", *parentID)
	} else {
		q = q.Where("parent_id IS NULL")
	}
	q.Select("COALESCE(MAX(sort_order), 0)").Scan(&max)
	return max.Float64 + 1
}

// isAncestor 判断 ancestorID 是否是 nodeID 的祖先。
func (s *Server) isAncestor(nodeID, ancestorID string) bool {
	current := &ancestorID
	for i := 0; i < 32 && current != nil; i++ { // 深度上限防环死循环
		if *current == nodeID {
			return true
		}
		var parent sql.NullString
		s.db.Model(&models.Node{}).Select("parent_id").Where("id = ?", *current).Scan(&parent)
		if !parent.Valid {
			return false
		}
		current = &parent.String
	}
	return false
}
