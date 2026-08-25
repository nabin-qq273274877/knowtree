package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/nabin-qq273274877/knowtree/internal/models"
)

const defaultUpdateURL = "https://api.github.com/repos/nabin-qq273274877/knowtree/releases/latest"

type releaseInfo struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name                string `json:"name"`
		BrowserDownloadURL  string `json:"browser_download_url"`
	} `json:"assets"`
}

func (s *Server) updateSource() string {
	var row models.Setting
	if err := s.db.First(&row, "`key` = 'update.base_url'").Error; err == nil && row.ValueJSON != "" {
		v := strings.Trim(row.ValueJSON, `"`)
		if v != "" {
			return strings.TrimRight(v, "/")
		}
	}
	return defaultUpdateURL
}

func fetchLatestRelease(source string) (*releaseInfo, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", source, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("更新源返回 %d", resp.StatusCode)
	}
	var rel releaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("更新源未返回版本号")
	}
	return &rel, nil
}

// isNewer 语义化版本比较（容忍前缀 v 与非数字后缀）。
func isNewer(latest, current string) bool {
	l := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(latest), "v"), "-", 2)[0]
	c := strings.SplitN(strings.TrimPrefix(strings.TrimSpace(current), "v"), "-", 2)[0]
	lp := strings.Split(l, ".")
	cp := strings.Split(c, ".")
	for i := 0; i < 3; i++ {
		li, ci := atoiSafe(getOr(lp, i, "0")), atoiSafe(getOr(cp, i, "0"))
		if li != ci {
			return li > ci
		}
	}
	return false // 相同版本不算有更新
}

func getOr(arr []string, i int, def string) string {
	if i < len(arr) {
		return arr[i]
	}
	return def
}

func atoiSafe(s string) int {
	n := 0
	for _, r := range s {
		if r < '0' || r > '9' {
			return n
		}
		n = n*10 + int(r-'0')
	}
	return n
}

func assetNameFor(goos, goarch string) string {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	return fmt.Sprintf("knowtree-%s-%s%s", goos, goarch, ext)
}

// ---- POST /api/update/check ----

func (s *Server) updateCheck(c *gin.Context) {
	rel, err := fetchLatestRelease(s.updateSource())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "检查更新失败：" + err.Error()})
		return
	}
	want := assetNameFor(runtime.GOOS, runtime.GOARCH)
	var assetURL string
	for _, a := range rel.Assets {
		if a.Name == want {
			assetURL = a.BrowserDownloadURL
			break
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"current":      s.cfg.Version,
		"latest":       rel.TagName,
		"has_update":   isNewer(rel.TagName, s.cfg.Version),
		"name":         rel.Name,
		"notes":        rel.Body,
		"asset_name":   want,
		"asset_exists": assetURL != "",
	})
}

// ---- POST /api/update/apply ----

type updateApplyResult struct {
	OK          bool   `json:"ok"`
	Version     string `json:"version,omitempty"`
	Message     string `json:"message,omitempty"`
	RestartHint bool   `json:"restart_hint"`
}

func (s *Server) updateApply(c *gin.Context) {
	rel, err := fetchLatestRelease(s.updateSource())
	if err != nil {
		c.JSON(http.StatusBadGateway, updateApplyResult{Message: "检查失败：" + err.Error()})
		return
	}
	if !isNewer(rel.TagName, s.cfg.Version) {
		c.JSON(http.StatusOK, updateApplyResult{OK: false, Version: rel.TagName, Message: "已是最新版本"})
		return
	}

	want := assetNameFor(runtime.GOOS, runtime.GOARCH)
	var binURL, sumURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case want:
			binURL = a.BrowserDownloadURL
		case "checksums.txt":
			sumURL = a.BrowserDownloadURL
		}
	}
	if binURL == "" {
		c.JSON(http.StatusBadGateway, updateApplyResult{Message: "更新源缺少本平台产物：" + want})
		return
	}

	client := &http.Client{Timeout: 10 * time.Minute}

	// 1. 下载新二进制
	tmpBin, err := os.CreateTemp("", "knowtree-update-*")
	if err != nil {
		c.JSON(http.StatusInternalServerError, updateApplyResult{Message: err.Error()})
		return
	}
	tmpPath := tmpBin.Name()
	defer os.Remove(tmpPath)
	if err := downloadTo(client, binURL, tmpBin); err != nil {
		tmpBin.Close()
		c.JSON(http.StatusBadGateway, updateApplyResult{Message: "下载失败：" + err.Error()})
		return
	}
	tmpBin.Close()

	// 2. SHA256 校验（对照 checksums.txt）
	if sumURL != "" {
		sumTmp, err := os.CreateTemp("", "knowtree-sum-*")
		if err == nil {
			sumPath := sumTmp.Name()
			defer os.Remove(sumPath)
			err = downloadTo(client, sumURL, sumTmp)
			sumTmp.Close()
			if err == nil {
				if matchErr := verifyChecksum(sumPath, want, tmpPath); matchErr != nil {
					c.JSON(http.StatusBadGateway, updateApplyResult{Message: "校验失败：" + matchErr.Error()})
					return
				}
			} else if matchErr := err; matchErr != nil {
				c.JSON(http.StatusBadGateway, updateApplyResult{Message: "校验文件下载失败，已中止更新以保安全。"})
				return
			}
		}
	} else {
		c.JSON(http.StatusBadGateway, updateApplyResult{Message: "更新源缺少 checksums.txt，已中止更新以保安全。"})
		return
	}

	// 3. 原子替换：先把运行中的 exe 改名（Windows 允许），再落新文件；失败回滚
	exePath, err := os.Executable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, updateApplyResult{Message: err.Error()})
		return
	}
	bakPath := exePath + ".bak"
	os.Remove(bakPath)
	if err := os.Rename(exePath, bakPath); err != nil {
		c.JSON(http.StatusInternalServerError, updateApplyResult{Message: "备份旧程序失败：" + err.Error()})
		return
	}
	if err := copyFile(tmpPath, exePath); err != nil {
		_ = os.Rename(bakPath, exePath) // 回滚
		c.JSON(http.StatusInternalServerError, updateApplyResult{Message: "替换程序失败（已回滚）：" + err.Error()})
		return
	}
	// 权限兜底（unix）
	chmodExec(exePath)

	c.JSON(http.StatusOK, updateApplyResult{OK: true, Version: rel.TagName, RestartHint: true,
		Message: "更新完成，重启后生效"})
}

// ---- POST /api/update/restart ----

func (s *Server) updateRestart(c *gin.Context) {
	exe, err := os.Executable()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "拉起新进程失败：" + err.Error()})
		return
	}
	go func() {
		time.Sleep(600 * time.Millisecond)
		os.Exit(0)
	}()
	c.JSON(http.StatusOK, gin.H{"restarting": true})
}

// ---- helpers ----

func downloadTo(client *http.Client, url string, f *os.File) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(checksumFile, assetName, downloaded string) error {
	data, err := os.ReadFile(checksumFile)
	if err != nil {
		return err
	}
	var expected string
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) >= 2 && fields[1] == assetName {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if expected == "" {
		return fmt.Errorf("checksums 中没有 %s", assetName)
	}
	f, err := os.Open(downloaded)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != expected {
		return fmt.Errorf("SHA256 不匹配（期望 %s，实际 %s）", expected[:12], got[:12])
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func chmodExec(path string) {
	if runtime.GOOS != "windows" {
		os.Chmod(path, 0o755)
	}
}
