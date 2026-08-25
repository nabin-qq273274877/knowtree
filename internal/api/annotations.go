package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

type saveAnnotationReq struct {
	ContentMd string `json:"content_md"`
}

func (s *Server) listAnnotations(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var items []models.Annotation
	if err := s.db.Where("node_id = ?", nodeID).Order("created_at DESC, id DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createAnnotation(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var req saveAnnotationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ContentMd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_md is required"})
		return
	}
	now := time.Now().Unix()
	a := models.Annotation{
		ID:        uuid.NewString(),
		NodeID:    nodeID,
		ContentMd: req.ContentMd,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.db.Create(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, a)
}

func (s *Server) updateAnnotation(c *gin.Context) {
	id := c.Param("id")
	var a models.Annotation
	if err := s.db.First(&a, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "annotation not found"})
		return
	}
	var req saveAnnotationReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ContentMd == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "content_md is required"})
		return
	}
	a.ContentMd = req.ContentMd
	a.UpdatedAt = time.Now().Unix()
	if err := s.db.Save(&a).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, a)
}

func (s *Server) deleteAnnotation(c *gin.Context) {
	id := c.Param("id")
	res := s.db.Delete(&models.Annotation{}, "id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "annotation not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
