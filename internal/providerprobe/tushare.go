package providerprobe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

// TushareConfig defines validation probes for Tushare Pro.
// TushareConfig 定义 Tushare Pro 的验证探针。
type TushareConfig struct {
	BaseURL string
	Token   string
	Timeout time.Duration
}

// ProbeTushare validates Tushare Pro response envelopes without wiring a business provider.
// ProbeTushare 只验证 Tushare Pro 响应 envelope，不接入业务 provider。
func ProbeTushare(ctx context.Context, cfg TushareConfig) Report {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "http://api.tushare.pro"
	}
	report := NewReport("tushare", "tushare_user_token_required")
	if cfg.Token == "" {
		report.Add(Check{Name: "token", Endpoint: cfg.BaseURL, Passed: false, Message: "TUSHARE_TOKEN is required for live validation"})
		return report
	}

	client := &http.Client{Timeout: cfg.Timeout}
	if cfg.Timeout <= 0 {
		client.Timeout = 20 * time.Second
	}

	probes := []struct {
		name     string
		apiName  string
		params   map[string]any
		fields   string
		required []string
	}{
		{
			name:     "fund_basic",
			apiName:  "fund_basic",
			params:   map[string]any{"market": "E"},
			fields:   "ts_code,name,management,custodian,fund_type,found_date,due_date,list_date,issue_amount,m_fee,c_fee",
			required: []string{"ts_code", "name", "fund_type"},
		},
		{
			name:     "fund_nav",
			apiName:  "fund_nav",
			params:   map[string]any{"ts_code": "000001.OF"},
			fields:   "ts_code,ann_date,end_date,unit_nav,accum_nav,net_asset,total_netasset",
			required: []string{"ts_code", "end_date", "unit_nav"},
		},
		{
			name:     "index_daily",
			apiName:  "index_daily",
			params:   map[string]any{"ts_code": "000300.SH"},
			fields:   "ts_code,trade_date,close,pct_chg",
			required: []string{"ts_code", "trade_date", "close"},
		},
	}

	for _, probe := range probes {
		check := runTushareProbe(ctx, client, cfg.BaseURL, cfg.Token, probe.name, probe.apiName, probe.params, probe.fields, probe.required)
		report.Add(check)
	}
	return report
}

func runTushareProbe(ctx context.Context, client *http.Client, baseURL, token, name, apiName string, params map[string]any, fields string, required []string) Check {
	check := Check{Name: name, Endpoint: baseURL + " api_name=" + apiName, RequiredFields: required}
	payload := map[string]any{
		"api_name": apiName,
		"token":    token,
		"params":   params,
		"fields":   fields,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		check.Message = err.Error()
		return check
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		check.Message = err.Error()
		return check
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		check.Message = err.Error()
		return check
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		check.Message = fmt.Sprintf("http status %d", resp.StatusCode)
		return check
	}
	var envelope struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			Fields []string `json:"fields"`
			Items  [][]any  `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		check.Message = err.Error()
		return check
	}
	check.ObservedFields = append([]string{}, envelope.Data.Fields...)
	sort.Strings(check.ObservedFields)
	if envelope.Code != 0 {
		check.Message = fmt.Sprintf("tushare code %d: %s", envelope.Code, envelope.Msg)
		return check
	}
	if len(envelope.Data.Items) == 0 {
		check.Message = "no sample rows returned"
		return check
	}
	fieldSet := map[string]bool{}
	for _, field := range envelope.Data.Fields {
		fieldSet[field] = true
	}
	var missing []string
	for _, field := range required {
		if !fieldSet[field] {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		check.Message = "missing fields: " + strings.Join(missing, ", ")
		return check
	}
	check.Passed = true
	return check
}
