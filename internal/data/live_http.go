// live_http.go centralizes bounded JSON transport and payload hashes for live providers.
// live_http.go 为实时 provider 统一有界 JSON 传输和载荷哈希。
package data

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// liveHTTPClient keeps bounded HTTP and raw-payload hashing consistent across providers.
// liveHTTPClient 在各 provider 间统一有界 HTTP 请求和原始载荷哈希。
type liveHTTPClient struct {
	client *http.Client
}

// newLiveHTTPClient creates a client with a defensive provider timeout.
// newLiveHTTPClient 创建带防御性 provider 超时的客户端。
func newLiveHTTPClient(client *http.Client) *liveHTTPClient {
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	return &liveHTTPClient{client: client}
}

// getJSON fetches and decodes one JSON object while retaining its content hash.
// getJSON 获取并解码一个 JSON 对象，同时保留内容哈希。
func (c *liveHTTPClient) getJSON(ctx context.Context, url string) (map[string]any, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	return c.doJSON(req)
}

// postJSON sends a JSON object and retains the response content hash.
// postJSON 发送 JSON 对象并保留响应内容哈希。
func (c *liveHTTPClient) postJSON(ctx context.Context, url string, payload any) (map[string]any, string, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.doJSON(req)
}

func (c *liveHTTPClient) doJSON(req *http.Request) (map[string]any, string, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, "", fmt.Errorf("provider returned HTTP %d", resp.StatusCode)
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, "", err
	}
	sum := sha256.Sum256(body)
	return payload, hex.EncodeToString(sum[:]), nil
}
