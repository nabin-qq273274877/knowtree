// Package llm 提供 OpenAI 兼容 Chat Completions 客户端（流式/非流式），
// 配置持久化于 settings 表（键前缀 llm.）。
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

// 设置表中的配置键
const (
	KeyBaseURL    = "llm.base_url"
	KeyAPIKey     = "llm.api_key"
	KeyModel      = "llm.model"
	KeyTemperature = "llm.temperature"
	KeyMaxTokens  = "llm.max_tokens"
)

type Config struct {
	BaseURL     string  `json:"base_url"`
	APIKey      string  `json:"api_key"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"max_tokens"`
}

// settingString 读取 settings 表里的字符串值：
// 值以 JSON 形式存储（json.Marshal），优先正规解码；
// 对历史脏数据（粘贴进来的引号/转义）做兜底剥离。
func settingString(raw string) string {
	var s string
	if err := json.Unmarshal([]byte(raw), &s); err == nil {
		s = strings.TrimSpace(s)
	} else {
		s = strings.TrimSpace(raw)
	}
	// 再兜底剥一层：用户可能把引号粘贴进值里
	return strings.Trim(s, "\\\"'`")
}

// Load 从 settings 表读取 LLM 配置；未配置返回 ErrNotConfigured。
func Load(db *gorm.DB) (Config, error) {
	var rows []models.Setting
	if err := db.Where("`key` LIKE ?", "llm.%").Find(&rows).Error; err != nil {
		return Config{}, err
	}
	cfg := Config{Temperature: 0.7, MaxTokens: 2048}
	for _, r := range rows {
		switch r.Key {
		case KeyBaseURL:
			cfg.BaseURL = settingString(r.ValueJSON)
		case KeyAPIKey:
			cfg.APIKey = settingString(r.ValueJSON)
		case KeyModel:
			cfg.Model = settingString(r.ValueJSON)
		case KeyTemperature:
			fmt.Sscanf(r.ValueJSON, "%f", &cfg.Temperature)
		case KeyMaxTokens:
			fmt.Sscanf(r.ValueJSON, "%d", &cfg.MaxTokens)
		}
	}
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}

var ErrNotConfigured = fmt.Errorf("LLM 未配置：请在「设置」中填写 Base URL、模型与 API Key")

func (c Config) Validate() error {
	if c.BaseURL == "" || c.Model == "" {
		return ErrNotConfigured
	}
	// 兜底清洗：引号/空白/多余斜杠
	c.BaseURL = strings.Trim(strings.TrimSpace(c.BaseURL), "\"'`")
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")
	return nil
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

var httpClient = &http.Client{Timeout: 180 * time.Second}

func (c Config) doRequest(ctx context.Context, body any) (*http.Response, error) {
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/chat/completions", bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var eb bytes.Buffer
		_, _ = eb.ReadFrom(resp.Body)
		msg := eb.String()
		// 尝试提取 error.message
		var ej struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(eb.Bytes(), &ej) == nil && ej.Error.Message != "" {
			msg = ej.Error.Message
		}
		return nil, fmt.Errorf("LLM 服务返回 %d：%s", resp.StatusCode, truncate(msg, 300))
	}
	return resp, nil
}

// Complete 非流式补全，返回首条回复文本。
func (c Config) Complete(ctx context.Context, messages []Message) (string, error) {
	resp, err := c.doRequest(ctx, chatRequest{
		Model: c.Model, Messages: messages,
		Temperature: c.Temperature, MaxTokens: c.MaxTokens, Stream: false,
	})
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var out struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", fmt.Errorf("LLM 返回为空")
	}
	return out.Choices[0].Message.Content, nil
}

// Stream 流式补全，每收到一段增量文本回调一次。
func (c Config) Stream(ctx context.Context, messages []Message, onDelta func(string)) error {
	resp, err := c.doRequest(ctx, chatRequest{
		Model: c.Model, Messages: messages,
		Temperature: c.Temperature, MaxTokens: c.MaxTokens, Stream: true,
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			return nil
		}
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		for _, ch := range chunk.Choices {
			if ch.Delta.Content != "" {
				onDelta(ch.Delta.Content)
			}
		}
	}
	return scanner.Err()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
