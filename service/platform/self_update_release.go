package platform

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// releaseAsset 是 GitHub Release 里的一个发布资产
type releaseAsset struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	// Digest 形如 sha256:<hex>，由 GitHub 依据上传内容计算，是自更新的校验锚点
	Digest string `json:"digest"`
	URL    string `json:"browser_download_url"`
}

// sha256 从 digest 字段里取出十六进制校验值，格式不符时返回空串（调用方按"缺少校验值"处理）
func (a releaseAsset) sha256() string {
	parts := strings.SplitN(a.Digest, ":", 2)
	if len(parts) != 2 || !strings.EqualFold(strings.TrimSpace(parts[0]), "sha256") {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(parts[1]))
}

// releaseInfo 是一次 Release 查询的结果
type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Body    string         `json:"body"`
	Assets  []releaseAsset `json:"assets"`

	// fetchedAt 本次查询成功的时间戳（秒）
	fetchedAt int64
}

// asset 按系统选择发布资产
func (r *releaseInfo) asset(goos string) (releaseAsset, bool) {
	name := assetName(goos)
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}

	return releaseAsset{}, false
}

// assetName 系统对应的发布资产名，与 .github/workflows/go.yml 保持一致
func assetName(goos string) string {
	if goos == constant.Darwin {
		return constant.UpdateAssetDarwin
	}

	return constant.UpdateAssetLinux
}

// Release 缓存：未认证的 GitHub API 限流为每小时 60 次/IP，前端每次进页面都会查询，
// 必须缓存；同一时刻只允许一个查询在飞，其余请求等待后复用结果。
var (
	releaseMu   sync.Mutex
	releaseData *releaseInfo
	releaseAt   time.Time
)

// fetchRelease 查询最新 Release，refresh 为 false 时命中缓存直接返回
func fetchRelease(refresh bool) (*releaseInfo, error) {
	releaseMu.Lock()
	defer releaseMu.Unlock()

	if !refresh && releaseData != nil && time.Since(releaseAt) < constant.UpdateCheckTTL {
		// 命中缓存时 checkedAt 保持为上次成功查询的时间，便于前端与排查区分"缓存"与"实时"
		logger.Logger.Debugf("版本检查命中缓存, tag: %s, checkedAt: %d", releaseData.TagName, releaseData.fetchedAt)
		return releaseData, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), constant.UpdateAPITimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, constant.GitHubReleaseAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "dmp/"+constant.Version)

	resp, err := newHTTPClient(constant.UpdateAPITimeout).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github api 返回 %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var info releaseInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&info); err != nil {
		return nil, err
	}
	info.fetchedAt = time.Now().Unix()

	releaseData = &info
	releaseAt = time.Now()
	logger.Logger.Debugf("版本检查已查询 GitHub, tag: %s, refresh: %t, checkedAt: %d", info.TagName, refresh, info.fetchedAt)

	return releaseData, nil
}

// compareVersion 比较两个 vX.Y.Z 形式的版本号，返回 -1 / 0 / 1。
// 解析不出三段数字时退化为字符串比较，保证任何输入都有确定结果。
func compareVersion(a, b string) int {
	av, aok := parseVersion(a)
	bv, bok := parseVersion(b)
	if !aok || !bok {
		switch {
		case a == b:
			return 0
		case a > b:
			return 1
		default:
			return -1
		}
	}

	for i := 0; i < 3; i++ {
		if av[i] == bv[i] {
			continue
		}
		if av[i] > bv[i] {
			return 1
		}

		return -1
	}

	return 0
}

// parseVersion 解析 vX.Y.Z，忽略 -rc1 / +build 之类的后缀
func parseVersion(v string) ([3]int, bool) {
	var out [3]int

	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return out, false
	}
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i]
	}

	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}

	return out, true
}

// inContainer 是否运行在容器里：Dockerfile 会设置 DMP_IN_CONTAINER=1，
// 同时兼容没有设置该变量的第三方镜像
func inContainer() bool {
	if os.Getenv("DMP_IN_CONTAINER") == "1" {
		return true
	}
	for _, path := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	return false
}

// selfUpdateBlock 判断当前环境能否在网页上自更新，返回禁用原因对应的错误码与原因标识。
// 返回 0 表示可以更新。
func selfUpdateBlock() (int, string) {
	if inContainer() {
		return constant.CodePlatformSelfUpdateContainer, SelfUpdateReasonContainer
	}
	// 发布流水线只产出 linux 与 darwin 的 amd64 资产
	if runtime.GOOS != "linux" && runtime.GOOS != constant.Darwin {
		return constant.CodePlatformSelfUpdateUnsupported, SelfUpdateReasonUnsupported
	}
	if runtime.GOARCH != "amd64" {
		return constant.CodePlatformSelfUpdateUnsupported, SelfUpdateReasonUnsupported
	}

	exe, err := currentBinary()
	if err != nil {
		return constant.CodePlatformSelfUpdateUnsupported, SelfUpdateReasonUnsupported
	}
	// go run 的产物在临时目录，替换它没有意义
	if strings.Contains(exe, string(filepath.Separator)+"go-build") || strings.HasPrefix(exe, os.TempDir()) {
		return constant.CodePlatformSelfUpdateUnsupported, SelfUpdateReasonUnsupported
	}
	info, err := os.Stat(exe)
	if err != nil || info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return constant.CodePlatformSelfUpdateUnsupported, SelfUpdateReasonUnsupported
	}
	if !dirWritable(filepath.Dir(exe)) {
		return constant.CodePlatformSelfUpdateNotWritable, SelfUpdateReasonNotWritable
	}

	return 0, SelfUpdateReasonReady
}

// currentBinary 当前可执行文件的绝对路径。
// 解析符号链接，因为替换的必须是真实文件——重命名符号链接只会换掉链接本身。
func currentBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	if resolved, evalErr := filepath.EvalSymlinks(exe); evalErr == nil {
		exe = resolved
	}

	return exe, nil
}

// dirWritable 目录是否可写：创建一个探针文件后立即删除
func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".dmp-update-probe")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)

	return true
}

// downloadAsset 下载安装包，返回下载内容的 sha256。
// 提供了加速节点时先走加速节点，失败后自动回退直连；
// 校验值始终取自 api.github.com 的 digest，与下载通道无关。
func downloadAsset(asset releaseAsset, proxy string) (string, error) {
	target := downloadPath()

	urls := []string{asset.URL}
	if proxy != "" {
		urls = []string{strings.TrimSuffix(proxy, "/") + "/" + asset.URL, asset.URL}
	}

	var lastErr error
	for i, u := range urls {
		if i > 0 {
			logger.Logger.Warnf("加速节点下载失败, 回退直连, err: %v", lastErr)
			cache.UpdateTask.Log("加速节点下载失败，改为直连下载")
		}

		sum, err := downloadTo(u, target, asset.Size)
		if err == nil {
			return sum, nil
		}
		lastErr = err
		_ = os.Remove(target)
	}

	return "", lastErr
}

// downloadTo 把 url 下载到 target，边写盘边计算 sha256 并上报进度
func downloadTo(url, target string, expectSize int64) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.UpdateDownloadTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "dmp/"+constant.Version)

	resp, err := newHTTPClient(0).Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载返回 %d", resp.StatusCode)
	}

	f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}

	total := resp.ContentLength
	if total <= 0 {
		total = expectSize
	}
	hash := sha256.New()
	progress := &downloadProgress{total: total}
	written, err := io.Copy(io.MultiWriter(f, hash, progress), io.LimitReader(resp.Body, constant.UpdateMaxAssetBytes+1))
	if err != nil {
		_ = f.Close()
		return "", err
	}
	if written > constant.UpdateMaxAssetBytes {
		_ = f.Close()
		return "", fmt.Errorf("安装包超过 %d 字节上限", int64(constant.UpdateMaxAssetBytes))
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// downloadProgress 把已下载字节数换算成任务进度（5%–55%），只在百分比变化时上报
type downloadProgress struct {
	total   int64
	written int64
	lastPct int
}

func (p *downloadProgress) Write(b []byte) (int, error) {
	p.written += int64(len(b))
	if p.total <= 0 {
		return len(b), nil
	}

	pct := int(p.written * 50 / p.total)
	if pct > p.lastPct {
		p.lastPct = pct
		cache.UpdateTask.Progress(cache.UpdatePhaseDownloading, 5+pct, "正在下载安装包")
	}

	return len(b), nil
}

// extractBinary 从安装包里取出可执行文件，写到 target 并赋予执行权限。
// target 与可执行文件同目录，保证后续替换是同目录 rename。
func extractBinary(tgzPath, target string) error {
	f, err := os.Open(tgzPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() {
		_ = gz.Close()
	}()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if !hdr.FileInfo().Mode().IsRegular() {
			continue
		}
		if filepath.Base(hdr.Name) != constant.UpdateBinaryName {
			continue
		}

		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, io.LimitReader(tr, constant.UpdateMaxAssetBytes)); err != nil {
			_ = out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		if err := os.Chmod(target, 0o755); err != nil {
			return err
		}

		return nil
	}

	return fmt.Errorf("安装包内没有找到 %s", constant.UpdateBinaryName)
}

// verifyBinary 预检新版本：执行 `<新文件> -v`，首行必须等于目标 tag。
// 这一步同时验证了"文件能执行"（架构正确）与"资产与 tag 一致"（不是错版包）。
func verifyBinary(path, tag string) error {
	ctx, cancel := context.WithTimeout(context.Background(), constant.UpdatePreflightTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "-v")
	cmd.Dir = filepath.Dir(path)
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("执行新版本 -v 失败: %w", err)
	}

	first := strings.TrimSpace(strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0])
	if first != tag {
		return fmt.Errorf("新版本输出 %q 与目标版本 %q 不一致", first, tag)
	}

	return nil
}

// newHTTPClient 构造自更新用的 HTTP 客户端。timeout 为 0 表示只由 context 控制
// （安装包可能几十 MB，固定短超时会中途失败）。
func newHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 15 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConns:          4,
		IdleConnTimeout:       60 * time.Second,
	}

	return &http.Client{Transport: transport, Timeout: timeout}
}
