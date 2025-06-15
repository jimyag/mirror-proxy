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

func (e *Executer) Handle(w http.ResponseWriter, r *http.Request) {
	clientIP := strings.Split(r.RemoteAddr, ":")[0]
	srcIP, err := netip.ParseAddr(clientIP)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid source ip: %s, %s", clientIP, err.Error()), http.StatusBadRequest)
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
	// 获取原始路径并解码
	targetHostPath := r.URL.Path
	// 去掉前缀 /
	targetHostPath = strings.TrimPrefix(targetHostPath, "/")

	if strings.HasPrefix(targetHostPath, "https:/") {
		targetHostPath = strings.TrimPrefix(targetHostPath, "https://")
		targetHostPath = strings.TrimPrefix(targetHostPath, "https:/")
		targetHostPath = "https://" + targetHostPath
	} else if strings.HasPrefix(targetHostPath, "http:/") {
		targetHostPath = strings.TrimPrefix(targetHostPath, "http://")
		targetHostPath = strings.TrimPrefix(targetHostPath, "http:/")
		targetHostPath = "http://" + targetHostPath
	}

	// 解析 URL
	targetURL, err := url.Parse(targetHostPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 设置查询参数
	targetURL.RawQuery = r.URL.RawQuery

	// 如果没有协议，设置为 https
	if targetURL.Scheme == "" {
		targetURL.Scheme = "https"
	}

	// 如果主机名为空，使用路径作为主机名
	if targetURL.Host == "" {
		// 分割路径，第一部分作为主机名，剩余部分作为路径
		parts := strings.SplitN(targetURL.Path, "/", 2)
		targetURL.Host = parts[0]
		if len(parts) > 1 {
			targetURL.Path = "/" + parts[1]
		} else {
			targetURL.Path = "/"
		}
	}

	slog.Info("final target url", "target_url", targetURL.String())
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
