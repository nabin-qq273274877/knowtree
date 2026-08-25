package models

// 与 migrations/00001_init.sql 一一对应。
// 时间统一存 Unix 秒（INTEGER），由服务层显式赋值，不依赖 ORM 魔法。

type Node struct {
	ID               string  `gorm:"column:id;primaryKey" json:"id"`
	Title            string  `gorm:"column:title" json:"title"`
	ContentMd        string  `gorm:"column:content_md" json:"content_md"`
	Status           string  `gorm:"column:status" json:"status"`
	Stage            *string `gorm:"column:stage" json:"stage"`
	SortOrder        float64 `gorm:"column:sort_order" json:"sort_order"`
	ParentID         *string `gorm:"column:parent_id" json:"parent_id"`
	PosX             *float64 `gorm:"column:pos_x" json:"pos_x"`
	PosY             *float64 `gorm:"column:pos_y" json:"pos_y"`
	StatusChangedAt  *int64  `gorm:"column:status_changed_at" json:"status_changed_at"`
	SourceNote       *string `gorm:"column:source_note" json:"source_note"`
	CreatedAt        int64   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt        int64   `gorm:"column:updated_at" json:"updated_at"`
}

func (Node) TableName() string { return "nodes" }

type Edge struct {
	ID        string  `gorm:"column:id;primaryKey" json:"id"`
	SourceID  string  `gorm:"column:source_id" json:"source_id"`
	TargetID  string  `gorm:"column:target_id" json:"target_id"`
	Relation  string  `gorm:"column:relation" json:"relation"`
	Label     *string `gorm:"column:label" json:"label"`
	CreatedAt int64   `gorm:"column:created_at" json:"created_at"`
}

func (Edge) TableName() string { return "edges" }

type Resource struct {
	ID        string  `gorm:"column:id;primaryKey" json:"id"`
	NodeID    string  `gorm:"column:node_id" json:"node_id"`
	Kind      string  `gorm:"column:kind" json:"kind"`
	Title     string  `gorm:"column:title" json:"title"`
	URL       *string `gorm:"column:url" json:"url"`
	Path      *string `gorm:"column:path" json:"path"`
	Note      *string `gorm:"column:note" json:"note"`
	CreatedAt int64   `gorm:"column:created_at" json:"created_at"`
}

func (Resource) TableName() string { return "resources" }

type Exercise struct {
	ID           string  `gorm:"column:id;primaryKey" json:"id"`
	NodeID       string  `gorm:"column:node_id" json:"node_id"`
	Type         string  `gorm:"column:type" json:"type"`
	QuestionMd   string  `gorm:"column:question_md" json:"question_md"`
	OptionsJSON  *string `gorm:"column:options_json" json:"options_json"`
	AnswerMd     string  `gorm:"column:answer_md" json:"answer_md"`
	AnalysisMd   *string `gorm:"column:analysis_md" json:"analysis_md"`
	Difficulty   *int    `gorm:"column:difficulty" json:"difficulty"`
	AnswerDraft  *string `gorm:"column:answer_draft" json:"answer_draft"`
	Result       *string `gorm:"column:result" json:"result"`
	Score        *int    `gorm:"column:score" json:"score"`
	FeedbackMd   *string `gorm:"column:feedback_md" json:"feedback_md"`
	AnsweredAt   *int64  `gorm:"column:answered_at" json:"answered_at"`
	CreatedAt    int64   `gorm:"column:created_at" json:"created_at"`
}

func (Exercise) TableName() string { return "exercises" }

type Annotation struct {
	ID        string  `gorm:"column:id;primaryKey" json:"id"`
	NodeID    string  `gorm:"column:node_id" json:"node_id"`
	ContentMd string  `gorm:"column:content_md" json:"content_md"`
	CreatedAt int64   `gorm:"column:created_at" json:"created_at"`
	UpdatedAt int64   `gorm:"column:updated_at" json:"updated_at"`
}

func (Annotation) TableName() string { return "annotations" }

type Setting struct {
	Key       string `gorm:"column:key;primaryKey" json:"key"`
	ValueJSON string `gorm:"column:value_json" json:"value_json"`
}

func (Setting) TableName() string { return "settings" }
