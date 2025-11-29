package core

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"
)

// NewHTTPClient 创建一个通用的 HTTP 客户端
// - timeoutSec: 超时时间（秒）
// - proxy: 代理地址，例如 "http://127.0.0.1:7890"，留空则不设置代理
// 注意：不要在本包复用/复制平台内的请求逻辑，平台可自由决定是否使用该构造器。
func NewHTTPClient(timeoutSec int, proxy string) *http.Client {
	if timeoutSec <= 0 {
		timeoutSec = 30
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		MinVersion:         tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig:       tlsConfig,
		TLSHandshakeTimeout:   30 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if proxy != "" {
		if proxyURL, err := url.Parse(proxy); err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	client := &http.Client{
		Timeout:   time.Duration(timeoutSec) * time.Second,
		Transport: transport,
	}

	return client
}
