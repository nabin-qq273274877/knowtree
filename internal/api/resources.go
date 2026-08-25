package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

type createResourceReq struct {
	Title string  `json:"title"`
	Kind  string  `json:"kind"` // link | file（file 为 P1，先占位校验）
	URL   *string `json:"url"`
	Note  *string `json:"note"`
}

func (s *Server) listResources(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var items []models.Resource
	if err := s.db.Where("node_id = ?", nodeID).Order("created_at ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

func (s *Server) createResource(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var req createResourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title is required"})
		return
	}
	if req.Kind == "" {
		req.Kind = "link"
	}
	if req.Kind != "link" && req.Kind != "file" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid kind"})
		return
	}
	if req.Kind == "link" && (req.URL == nil || *req.URL == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "url is required for link resource"})
		return
	}
	r := models.Resource{
		ID:        uuid.NewString(),
		NodeID:    nodeID,
		Kind:      req.Kind,
		Title:     req.Title,
		URL:       req.URL,
		Note:      req.Note,
		CreatedAt: time.Now().Unix(),
	}
	if err := s.db.Create(&r).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (s *Server) deleteResource(c *gin.Context) {
	id := c.Param("id")
	res := s.db.Delete(&models.Resource{}, "id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}
