package providerprobe

import "time"

// Report records validation evidence for a real data source before provider coding.
// Report 在 provider 编码前记录真实数据源验证证据。
type Report struct {
	Provider     string    `json:"provider"`
	GeneratedAt  time.Time `json:"generated_at"`
	Passed       bool      `json:"passed"`
	LicenseTerms string    `json:"license_terms"`
	Checks       []Check   `json:"checks"`
}

// Check records one endpoint probe and the observed response shape.
// Check 记录一次 endpoint 探针及观察到的响应结构。
type Check struct {
	Name           string   `json:"name"`
	Endpoint       string   `json:"endpoint"`
	Passed         bool     `json:"passed"`
	Message        string   `json:"message"`
	RequiredFields []string `json:"required_fields,omitempty"`
	ObservedFields []string `json:"observed_fields,omitempty"`
}

// Add appends one check and updates the report status.
// Add 追加一个检查项并更新报告状态。
func (r *Report) Add(check Check) {
	if check.Message == "" && check.Passed {
		check.Message = "ok"
	}
	if check.Message == "" && !check.Passed {
		check.Message = "validation failed"
	}
	r.Checks = append(r.Checks, check)
	if !check.Passed {
		r.Passed = false
	}
}

// NewReport creates a report that starts in pass state until a check fails.
// NewReport 创建默认通过的报告，直到某项检查失败。
func NewReport(provider, licenseTerms string) Report {
	return Report{
		Provider:     provider,
		GeneratedAt:  time.Now().UTC(),
		Passed:       true,
		LicenseTerms: licenseTerms,
	}
}
