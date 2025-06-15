package execute

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/jimyag/mirror-proxy/config"
	"github.com/jimyag/mirror-proxy/constant"
	"github.com/jimyag/mirror-proxy/rules"
)

type Executer struct {
	rules []rules.Rule
}

func NewExecuter(cfg config.Config) (*Executer, error) {
	rs := make([]rules.Rule, 0, len(cfg.Rules))
	for _, r := range cfg.Rules {
		rule, err := rules.ParseRule(r, cfg)
		if err != nil {
			slog.With(
				slog.String("rule", r),
				slog.String("error", err.Error()),
			).Error("failed to parse rule")
			return nil, err
		}
		rs = append(rs, rule)
	}
	exe := Executer{
		rules: rs,
	}
	return &exe, nil
}

// getSrcIP 获取请求的真实来源 IP 地址
// 按照以下优先级获取：
// 1. X-Forwarded-For 头中的第一个 IP
// 2. X-Real-IP 头
// 3. RemoteAddr
func getSrcIP(r *http.Request) (netip.Addr, error) {
	// 1. 检查 X-Forwarded-For
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// 获取第一个 IP（最接近客户端的 IP）
		ips := strings.Split(forwardedFor, ",")
		if len(ips) > 0 {
			// 清理 IP 地址中的空格
			ip := strings.TrimSpace(ips[0])
			if addr, err := netip.ParseAddr(ip); err == nil {
				return addr, nil
			}
		}
	}

	// 2. 检查 X-Real-IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		if addr, err := netip.ParseAddr(realIP); err == nil {
			return addr, nil
		}
	}

	// 3. 使用 RemoteAddr
	// 处理 IPv6 地址的端口格式 [::1]:1234
	remoteAddr := r.RemoteAddr
	if strings.Contains(remoteAddr, "[") {
		// IPv6 地址格式
		lastColon := strings.LastIndex(remoteAddr, ":")
		if lastColon != -1 {
			remoteAddr = remoteAddr[:lastColon]
		}
		// 移除 IPv6 地址的方括号
		remoteAddr = strings.Trim(remoteAddr, "[]")
	} else {
		// IPv4 地址格式
		remoteAddr = strings.Split(remoteAddr, ":")[0]
	}

	addr, err := netip.ParseAddr(remoteAddr)
	if err != nil {
		return netip.Addr{}, err
	}

	return addr, nil
}

func (e *Executer) Handle(w http.ResponseWriter, r *http.Request) {
	srcIP, err := getSrcIP(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid source ip: %s", err.Error()), http.StatusBadRequest)
		return
	}

	targetURL, err := getTargetURL(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid target url: %s", err.Error()), http.StatusBadRequest)
		return
	}
	metadata := constant.Metadata{
		SrcIP:    srcIP,
		Host:     targetURL.Host,
		Protocol: targetURL.Scheme,
	}
	for _, rule := range e.rules {
		if rule.Match(metadata) {
			if rule.Action() == constant.RuleActionAllow {
				e.Execute(w, r, targetURL)
				return
			} else if rule.Action() == constant.RuleActionDeny {
				http.Error(w, "request denied", http.StatusForbidden)
				return
			}
		}
	}
	http.Error(w, "no rule matched", http.StatusBadRequest)
}

func getTargetURL(r *http.Request) (*url.URL, error) {
	// 获取并清理路径
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		return nil, fmt.Errorf("empty path")
	}

	// 处理 URL 格式
	var targetURLStr string
	switch {
	case strings.HasPrefix(path, "https://"):
		targetURLStr = path
	case strings.HasPrefix(path, "http://"):
		targetURLStr = path
	case strings.HasPrefix(path, "https:/"):
		targetURLStr = "https://" + strings.TrimPrefix(path, "https:/")
	case strings.HasPrefix(path, "http:/"):
		targetURLStr = "http://" + strings.TrimPrefix(path, "http:/")
	default:
		// 无协议的情况，默认使用 https
		targetURLStr = "https://" + path
	}

	// 解析 URL
	targetURL, err := url.Parse(targetURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// 验证 URL 格式
	if targetURL.Scheme == "" {
		return nil, fmt.Errorf("missing URL scheme")
	}
	if targetURL.Host == "" {
		return nil, fmt.Errorf("missing URL host")
	}

	// 处理查询参数
	if r.URL.RawQuery != "" {
		targetURL.RawQuery = r.URL.RawQuery
	}

	// 确保路径以 / 开头
	if !strings.HasPrefix(targetURL.Path, "/") {
		targetURL.Path = "/" + targetURL.Path
	}

	slog.Info("target URL processed",
		"original_path", r.URL.Path,
		"final_url", targetURL.String(),
	)

	return targetURL, nil
}

func (e *Executer) Execute(w http.ResponseWriter, r *http.Request, targetURL *url.URL) {
	now := time.Now()
	// 复制请求
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL.String(), r.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to create request: %s", err.Error())
		slog.With(
			slog.String("target_url", targetURL.String()),
		).Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	for key, values := range r.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Host = targetURL.Host

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		msg := fmt.Sprintf("upstream fetch failed: %s", err.Error())
		slog.With(
			slog.String("target_url", targetURL.String()),
		).Error(msg)
		http.Error(w, msg, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, v := range values {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(resp.StatusCode)

	n, err := io.Copy(w, resp.Body)
	if err != nil {
		msg := fmt.Sprintf("failed to copy response body: %s", err.Error())
		slog.With(
			slog.String("target_url", targetURL.String()),
		).Error(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	sizeMB := float64(n) / 1024 / 1024
	slog.With(
		slog.String("src_ip", r.RemoteAddr),
		slog.String("target_url", targetURL.String()),
		slog.Int("status_code", resp.StatusCode),
		slog.String("duration", time.Since(now).String()),
		slog.String("size_mb", fmt.Sprintf("%.2f", sizeMB)),
	).Info("request finished")
}
