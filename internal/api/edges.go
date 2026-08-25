package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

var validRelations = map[string]bool{
	"prerequisite": true,
	"related":      true,
}

type createEdgeReq struct {
	SourceID string  `json:"source_id"`
	TargetID string  `json:"target_id"`
	Relation string  `json:"relation"` // 缺省 related
	Label    *string `json:"label"`
}

type updateEdgeReq struct {
	SourceID *string `json:"source_id"`
	TargetID *string `json:"target_id"`
	Relation *string `json:"relation"`
	Label    *string `json:"label"`
}

func (s *Server) listEdges(c *gin.Context) {
	var edges []models.Edge
	if err := s.db.Order("created_at ASC").Find(&edges).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, edges)
}

func (s *Server) createEdge(c *gin.Context) {
	var req createEdgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Relation == "" {
		req.Relation = "related"
	}
	if !validRelations[req.Relation] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relation"})
		return
	}
	if req.SourceID == req.TargetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "self loop is not allowed"})
		return
	}
	if !s.nodeExists(&req.SourceID) || !s.nodeExists(&req.TargetID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source or target node not found"})
		return
	}
	if e := s.duplicateEdge(req.SourceID, req.TargetID, req.Relation, ""); e {
		c.JSON(http.StatusConflict, gin.H{"error": "edge already exists"})
		return
	}
	e := models.Edge{
		ID:        uuid.NewString(),
		SourceID:  req.SourceID,
		TargetID:  req.TargetID,
		Relation:  req.Relation,
		Label:     req.Label,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.db.Create(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, e)
}

func (s *Server) updateEdge(c *gin.Context) {
	id := c.Param("id")
	var e models.Edge
	if err := s.db.First(&e, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "edge not found"})
		return
	}
	var req updateEdgeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.SourceID != nil {
		e.SourceID = *req.SourceID
	}
	if req.TargetID != nil {
		e.TargetID = *req.TargetID
	}
	if e.SourceID == e.TargetID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "self loop is not allowed"})
		return
	}
	if !s.nodeExists(&e.SourceID) || !s.nodeExists(&e.TargetID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source or target node not found"})
		return
	}
	if req.Relation != nil {
		if !validRelations[*req.Relation] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid relation"})
			return
		}
		e.Relation = *req.Relation
	}
	if req.Label != nil {
		e.Label = req.Label
	}
	if s.duplicateEdge(e.SourceID, e.TargetID, e.Relation, id) {
		c.JSON(http.StatusConflict, gin.H{"error": "edge already exists"})
		return
	}
	if err := s.db.Save(&e).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, e)
}

func (s *Server) deleteEdge(c *gin.Context) {
	id := c.Param("id")
	res := s.db.Delete(&models.Edge{}, "id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "edge not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

// duplicateEdge 检查同源同目标同类型的连线是否已存在（excludeID 用于更新场景排除自身）。
func (s *Server) duplicateEdge(sourceID, targetID, relation, excludeID string) bool {
	var cnt int64
	q := s.db.Model(&models.Edge{}).
		Where("source_id = ? AND target_id = ? AND relation = ?", sourceID, targetID, relation)
	if excludeID != "" {
		q = q.Where("id <> ?", excludeID)
	}
	q.Count(&cnt)
	return cnt > 0
}
