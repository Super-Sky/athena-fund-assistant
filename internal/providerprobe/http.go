package providerprobe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"
)

// JSONClient fetches JSON endpoints with a bounded timeout.
// JSONClient 使用有界超时获取 JSON endpoint。
type JSONClient struct {
	client *http.Client
}

// NewJSONClient creates a JSON HTTP client.
// NewJSONClient 创建 JSON HTTP 客户端。
func NewJSONClient(timeout time.Duration) *JSONClient {
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	return &JSONClient{client: &http.Client{Timeout: timeout}}
}

// Get fetches one JSON document into a generic map.
// Get 获取一个 JSON 文档并解析为通用 map。
func (c *JSONClient) Get(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http status %d", resp.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func observedFields(payload map[string]any) []string {
	fields := make([]string, 0, len(payload))
	for key := range payload {
		fields = append(fields, key)
	}
	sort.Strings(fields)
	return fields
}

func missingFields(payload map[string]any, required []string) []string {
	var missing []string
	for _, field := range required {
		if _, ok := payload[field]; !ok {
			missing = append(missing, field)
		}
	}
	return missing
}
