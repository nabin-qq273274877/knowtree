package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nabin-qq273274877/knowtree/internal/llm"
	"github.com/nabin-qq273274877/knowtree/internal/models"
)

// ---- DTO ----

type exerciseItem struct {
	Type        string          `json:"type"`
	QuestionMd  string          `json:"question_md"`
	Options     json.RawMessage `json:"options,omitempty"`
	AnswerMd    string          `json:"answer_md"`
	AnalysisMd  *string         `json:"analysis_md,omitempty"`
	Difficulty  *int            `json:"difficulty,omitempty"`
}

type saveExercisesReq struct {
	Items []exerciseItem `json:"items"`
}

type saveDraftReq struct {
	AnswerDraft *string `json:"answer_draft"`
}

type submitReq struct {
	Answer string `json:"answer"`
}

// ---- GET /api/nodes/:id/exercises ----

func (s *Server) listExercises(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var items []models.Exercise
	if err := s.db.Where("node_id = ?", nodeID).Order("created_at ASC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, items)
}

// ---- POST /api/nodes/:id/exercises （批量保存生成的习题）----

func (s *Server) createExercises(c *gin.Context) {
	nodeID := c.Param("id")
	if !s.nodeExists(&nodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	var req saveExercisesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items is empty"})
		return
	}
	now := time.Now().Unix()
	created := make([]models.Exercise, 0, len(req.Items))
	for _, it := range req.Items {
		if !validExTypes[it.Type] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid type: " + it.Type})
			return
		}
		if it.QuestionMd == "" || it.AnswerMd == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "question_md and answer_md are required"})
			return
		}
		e := models.Exercise{
			ID:          uuid.NewString(),
			NodeID:      nodeID,
			Type:        it.Type,
			QuestionMd:  it.QuestionMd,
			AnswerMd:    it.AnswerMd,
			Difficulty:  it.Difficulty,
			CreatedAt:   now,
		}
		if len(it.Options) > 0 && string(it.Options) != "null" {
			opts := string(it.Options)
			e.OptionsJSON = &opts
		}
		if it.AnalysisMd != nil {
			e.AnalysisMd = it.AnalysisMd
		}
		created = append(created, e)
	}
	if err := s.db.Create(&created).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, created)
}

// ---- PATCH /api/exercises/:id （保存作答草稿）----

func (s *Server) updateExercise(c *gin.Context) {
	id := c.Param("id")
	var e models.Exercise
	if err := s.db.First(&e, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exercise not found"})
		return
	}
	var req saveDraftReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	e.AnswerDraft = req.AnswerDraft
	if err := s.db.Model(&models.Exercise{}).Where("id = ?", id).
		Update("answer_draft", req.AnswerDraft).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, e)
}

// ---- DELETE /api/exercises/:id ----

func (s *Server) deleteExercise(c *gin.Context) {
	id := c.Param("id")
	res := s.db.Delete(&models.Exercise{}, "id = ?", id)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": res.Error.Error()})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "exercise not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": res.RowsAffected})
}

// ---- POST /api/exercises/:id/submit （提交并批改，FR-9）----

type llmGradeOut struct {
	Result   string `json:"result"`
	Score    int    `json:"score"`
	Feedback string `json:"feedback"`
}

func (s *Server) submitExercise(c *gin.Context) {
	id := c.Param("id")
	var e models.Exercise
	if err := s.db.First(&e, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "exercise not found"})
		return
	}
	var req submitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	answer := strings.TrimSpace(req.Answer)
	if answer == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "answer is required"})
		return
	}

	now := time.Now().Unix()
	e.AnswerDraft = &answer
	e.AnsweredAt = &now

	switch e.Type {
	case "single_choice":
		if normalizeChoice(answer) == normalizeChoice(e.AnswerMd) {
			setResult(&e, "right", 100, "回答正确。")
		} else {
			setResult(&e, "wrong", 0, "回答错误，正确答案是 "+strings.ToUpper(normalizeChoice(e.AnswerMd))+"。")
		}
	case "multiple_choice":
		if normalizeChoice(answer) == normalizeChoice(e.AnswerMd) {
			setResult(&e, "right", 100, "回答正确。")
		} else {
			setResult(&e, "wrong", 0, "回答错误，正确答案是 "+normalizeChoice(e.AnswerMd)+".")
		}
	case "true_false":
		if normalizeTF(answer) == normalizeTF(e.AnswerMd) {
			setResult(&e, "right", 100, "回答正确。")
		} else {
			setResult(&e, "wrong", 0, "回答错误，正确答案是 "+normalizeTF(e.AnswerMd)+"。")
		}
	default: // fill_blank / short_answer → LLM 批改
		cfg, err := llm.Load(s.db)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "主观题批改需要 LLM：" + err.Error()})
			return
		}
		prompt := fmt.Sprintf(`请批改这道%s。
【题干】%s
【参考答案】%s
【学生作答】%s

判定标准：答对核心要点为 right；部分正确为 partial；完全错误或未答到点为 wrong。
只输出 JSON：{"result":"right|partial|wrong","score":0到100的整数,"feedback":"给学生的点评，先肯定再指正，100字内"}`,
			exTypeName(e.Type), e.QuestionMd, e.AnswerMd, answer)
		var g llmGradeOut
		if err := callJSON(c, cfg, prompt, &g); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "LLM 批改失败：" + err.Error()})
			return
		}
		if g.Result != "right" && g.Result != "partial" && g.Result != "wrong" {
			g.Result = "wrong"
		}
		if g.Score < 0 {
			g.Score = 0
		}
		fb := g.Feedback
		setResult(&e, g.Result, g.Score, fb)
	}

	if err := s.db.Model(&models.Exercise{}).Where("id = ?", e.ID).Updates(map[string]any{
		"answer_draft": e.AnswerDraft,
		"result":       e.Result,
		"score":        e.Score,
		"feedback_md":  e.FeedbackMd,
		"answered_at":  e.AnsweredAt,
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	s.db.First(&e, "id = ?", e.ID)
	c.JSON(http.StatusOK, e)
}

func setResult(e *models.Exercise, result string, score int, feedback string) {
	e.Result = &result
	e.Score = &score
	e.FeedbackMd = &feedback
}

func exTypeName(t string) string {
	m := map[string]string{
		"single_choice": "单选题", "multiple_choice": "多选题", "true_false": "判断题",
		"fill_blank": "填空题", "short_answer": "简答题",
	}
	return m[t]
}

// normalizeChoice 归一化选择题答案：大写、去空格与分隔符、按字母排序（多选）。
func normalizeChoice(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	var letters []rune
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			letters = append(letters, r)
		}
	}
	sort.Slice(letters, func(i, j int) bool { return letters[i] < letters[j] })
	return string(letters)
}

// normalizeTF 归一化判断题答案为「正确」/「错误」。
func normalizeTF(s string) string {
	t := strings.ToLower(strings.TrimSpace(s))
	truthy := map[string]bool{"正确": true, "对": true, "√": true, "t": true, "true": true, "yes": true, "y": true, "是": true}
	if truthy[t] {
		return "正确"
	}
	return "错误"
}
