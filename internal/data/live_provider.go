// live_provider.go defines the common credential-backed provider capability boundary.
// live_provider.go 定义需要用户凭据的 provider 通用能力边界。
package data

import (
	"context"
	"errors"
)

// ErrUnsupportedCapability makes missing provider coverage explicit to callers.
// ErrUnsupportedCapability 让调用方明确感知 provider 尚未覆盖的能力。
var ErrUnsupportedCapability = errors.New("market data capability is not supported by this provider")

// LiveProvider records the common contract for credential-backed data adapters.
// LiveProvider 记录需要用户凭据的实时数据适配器通用契约。
type LiveProvider interface {
	Provider
	ProviderName() string
	ValidateCredentials(context.Context) error
}
