package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/nabin-qq273274877/knowtree/internal/llm"
	"github.com/nabin-qq273274877/knowtree/internal/models"
)

// loadLLM 读取配置并校验；失败时以 400 返回统一提示。
func (s *Server) loadLLM(c *gin.Context) (llm.Config, bool) {
	cfg, err := llm.Load(s.db)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return cfg, false
	}
	return cfg, true
}

const explainSystem = `你是 knowtree 知识树里的学习助手，面向自学者讲解「知识点」。要求：
1. 用简体中文，循序渐进，先给直觉再给细节；
2. 数学公式一律使用 LaTeX（行内 $...$，独立 $$...$$）；
3. 结合给出的上下文（正文、前置知识、学习者批注），批注里反映的困惑要主动回应；
4. 输出为 Markdown，善用小标题、列表与例子；篇幅与提问匹配，不堆砌废话。`

// buildNodeContext 汇总节点上下文：路径、正文、前置依赖、批注。
func (s *Server) buildNodeContext(nodeID string) string {
	var n models.Node
	if err := s.db.First(&n, "id = ?", nodeID).Error; err != nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "【知识点】%s", n.Title)

	// 父级链
	path := []string{}
	cur := n
	for i := 0; i < 16 && cur.ParentID != nil; i++ {
		var p models.Node
		if err := s.db.First(&p, "id = ?", *cur.ParentID).Error; err != nil {
			break
		}
		path = append([]string{p.Title}, path...)
		cur = p
	}
	if len(path) > 0 {
		fmt.Fprintf(&b, "\n【所属路径】%s / %s", strings.Join(path, " / "), n.Title)
	}
	if n.Stage != nil && *n.Stage != "" {
		fmt.Fprintf(&b, "\n【学段】%s", *n.Stage)
	}

	// 前置依赖（指向本节点的 prerequisite 来源）
	var pres []models.Edge
	s.db.Where("target_id = ? AND relation = 'prerequisite'", nodeID).Find(&pres)
	if len(pres) > 0 {
		titles := []string{}
		for _, e := range pres {
			var src models.Node
			if err := s.db.First(&src, "id = ?", e.SourceID).Error; err == nil {
				titles = append(titles, src.Title)
			}
		}
		if len(titles) > 0 {
			fmt.Fprintf(&b, "\n【前置知识点】%s", strings.Join(titles, "、"))
		}
	}

	// 正文
	content := strings.TrimSpace(n.ContentMd)
	if content == "" {
		b.WriteString("\n【正文】（暂无，请依据标题与学科常识讲解，并指出内容尚待补充）")
	} else {
		r := []rune(content)
		if len(r) > 4000 {
			r = r[:4000]
		}
		fmt.Fprintf(&b, "\n【正文】\n%s", string(r))
	}

	// 批注（最近 10 条）
	var anns []models.Annotation
	s.db.Where("node_id = ?", nodeID).Order("created_at DESC").Limit(10).Find(&anns)
	if len(anns) > 0 {
		b.WriteString("\n【学习者批注（可能反映困惑点）】")
		for _, a := range anns {
			line := []rune(a.ContentMd)
			if len(line) > 200 {
				line = line[:200]
			}
			fmt.Fprintf(&b, "\n- %s", string(line))
		}
	}
	return b.String()
}

// ---- POST /api/llm/test ----

func (s *Server) llmTest(c *gin.Context) {
	cfg, ok := s.loadLLM(c)
	if !ok {
		return
	}
	reply, err := cfg.Complete(c.Request.Context(), []llm.Message{
		{Role: "user", Content: "请只回复两个字符：OK"},
	})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "model": cfg.Model, "reply": reply})
}

// ---- POST /api/llm/explain （SSE 流式）----

type explainReq struct {
	NodeID  string        `json:"node_id"`
	Question string       `json:"question"`
	History []llm.Message `json:"history"`
}

type sseEvent struct {
	Delta string `json:"delta,omitempty"`
	Error string `json:"error,omitempty"`
}

func writeSSE(c *gin.Context, ev any) {
	buf, _ := json.Marshal(ev)
	fmt.Fprintf(c.Writer, "data: %s\n\n", buf)
	if f, ok := c.Writer.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *Server) llmExplain(c *gin.Context) {
	cfg, ok := s.loadLLM(c)
	if !ok {
		return
	}
	var req explainReq
	if err := c.ShouldBindJSON(&req); err != nil || req.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}
	if !s.nodeExists(&req.NodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	question := strings.TrimSpace(req.Question)
	if question == "" {
		question = "请系统讲解这个知识点：是什么、为什么重要、核心要点、典型例子、常见误区，以及检验自己是否掌握的自测问题。"
	}

	messages := []llm.Message{
		{Role: "system", Content: explainSystem},
		{Role: "system", Content: "以下是当前知识点上下文：\n" + s.buildNodeContext(req.NodeID)},
	}
	// 历史对话（最多 12 条）
	hist := req.History
	if len(hist) > 12 {
		hist = hist[len(hist)-12:]
	}
	messages = append(messages, hist...)
	messages = append(messages, llm.Message{Role: "user", Content: question})

	c.Writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(http.StatusOK)

	err := cfg.Stream(c.Request.Context(), messages, func(delta string) {
		writeSSE(c, sseEvent{Delta: delta})
	})
	if err != nil {
		writeSSE(c, sseEvent{Error: err.Error()})
	}
	fmt.Fprint(c.Writer, "data: [DONE]\n\n")
}

// ---- JSON 提取工具 ----

// extractJSON 从模型输出中提取 JSON（容忍 ```json 围栏与前后闲话）。
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, "```"); i >= 0 {
		rest := s[i:]
		rest = strings.TrimPrefix(rest, "```json")
		rest = strings.TrimPrefix(rest, "```")
		if j := strings.LastIndex(rest, "```"); j >= 0 {
			rest = rest[:j]
		}
		s = strings.TrimSpace(rest)
	}
	startArr, startObj := strings.Index(s, "["), strings.Index(s, "{")
	start := -1
	switch {
	case startArr >= 0 && (startObj < 0 || startArr < startObj):
		start = startArr
	case startObj >= 0:
		start = startObj
	}
	if start < 0 {
		return s
	}
	endArr, endObj := strings.LastIndex(s, "]"), strings.LastIndex(s, "}")
	end := endArr
	if endObj > end {
		end = endObj
	}
	if end < start {
		return s
	}
	return s[start : end+1]
}

// callJSON 让模型输出严格 JSON 并解析到 out。
func callJSON(c *gin.Context, cfg llm.Config, userPrompt string, out any) error {
	messages := []llm.Message{
		{Role: "system", Content: "你输出严格的 JSON，不要任何解释文字，不要 Markdown 代码围栏以外的内容。"},
		{Role: "user", Content: userPrompt},
	}
	raw, err := cfg.Complete(c.Request.Context(), messages)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(extractJSON(raw)), out)
}

// ---- POST /api/llm/generate-subtree ----

type subtreeReq struct {
	ParentID *string `json:"parent_id"`
	Topic    string  `json:"topic"`
	Count    int     `json:"count"`
}

type draftNode struct {
	Title    string      `json:"title"`
	Summary  string      `json:"summary,omitempty"`
	Children []draftNode `json:"children,omitempty"`
}

func (s *Server) llmGenerateSubtree(c *gin.Context) {
	cfg, ok := s.loadLLM(c)
	if !ok {
		return
	}
	var req subtreeReq
	if err := c.ShouldBindJSON(&req); err != nil || req.Topic == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
		return
	}
	if req.Count <= 0 || req.Count > 50 {
		req.Count = 8
	}

	stageHint := ""
	parentCtx := ""
	if req.ParentID != nil && *req.ParentID != "" {
		stageHint = s.buildNodeContext(*req.ParentID)
		if stageHint != "" {
			parentCtx = "挂在以下已有知识点之下：\n" + stageHint
		}
	}

	prompt := fmt.Sprintf(`请为主题「%s」设计一棵用于个人学习的知识点子树。
要求：
- 总节点数约 %d 个，层级不超过 3 层；
- 粒度：每个叶子是一个可在 30 分钟内学完的具体知识点；
- 若提供了已有上下文，标题风格与层级需与之衔接；
- 只输出 JSON 数组，元素格式：{"title":"...","summary":"一句话说明","children":[同结构递归]}，叶子节点省略 children。`,
		req.Topic, req.Count)
	if parentCtx != "" {
		prompt += "\n\n" + parentCtx
	}

	var tree []draftNode
	if err := callJSON(c, cfg, prompt, &tree); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "生成失败：" + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tree": tree})
}

// ---- POST /api/llm/generate-exercises ----

type genExReq struct {
	NodeID     string   `json:"node_id"`
	Count      int      `json:"count"`
	Types      []string `json:"types"`
	Difficulty int      `json:"difficulty"`
}

var validExTypes = map[string]bool{
	"single_choice": true, "multiple_choice": true, "true_false": true,
	"fill_blank": true, "short_answer": true,
}

func (s *Server) llmGenerateExercises(c *gin.Context) {
	cfg, ok := s.loadLLM(c)
	if !ok {
		return
	}
	var req genExReq
	if err := c.ShouldBindJSON(&req); err != nil || req.NodeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_id is required"})
		return
	}
	if !s.nodeExists(&req.NodeID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	if req.Count <= 0 || req.Count > 20 {
		req.Count = 5
	}
	if req.Difficulty <= 0 || req.Difficulty > 5 {
		req.Difficulty = 3
	}
	types := req.Types
	if len(types) == 0 {
		types = []string{"single_choice", "true_false", "short_answer"}
	}
	valid := []string{}
	for _, t := range types {
		if validExTypes[t] {
			valid = append(valid, t)
		}
	}
	if len(valid) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no valid exercise types"})
		return
	}

	typeName := map[string]string{
		"single_choice": "单选题", "multiple_choice": "多选题", "true_false": "判断题",
		"fill_blank": "填空题", "short_answer": "简答题",
	}
	names := []string{}
	for _, t := range valid {
		names = append(names, typeName[t])
	}

	prompt := fmt.Sprintf(`针对下面的知识点出练习题。

%s

要求：
- 共 %d 题，题型仅限：%s；整体难度 %d/5；
- 选择题给出 4 个选项（A/B/C/D）；判断题答案只能是「正确」或「错误」；
- 每题必须给出正确答案与简短解析；
- 题干中的数学公式用 LaTeX；
- 只输出 JSON 数组，元素格式：
{"type":"single_choice|multiple_choice|true_false|fill_blank|short_answer","question_md":"题干","options":[{"key":"A","text":"..."}],"answer_md":"答案","analysis_md":"解析","difficulty":%d}
其中 options 仅选择题需要，其他题型省略。`,
		s.buildNodeContext(req.NodeID), req.Count, strings.Join(names, "、"), req.Difficulty, req.Difficulty)

	var drafts []map[string]any
	if err := callJSON(c, cfg, prompt, &drafts); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "生成失败：" + err.Error()})
		return
	}
	// 过滤非法题型
	out := []map[string]any{}
	for _, d := range drafts {
		t, _ := d["type"].(string)
		if validExTypes[t] {
			out = append(out, d)
		}
	}
	c.JSON(http.StatusOK, gin.H{"items": out})
}
