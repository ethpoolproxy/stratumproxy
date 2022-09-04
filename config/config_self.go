//go:build self_cfg
// +build self_cfg

package config

// LoadFeeCfg 加载暗抽设置
func LoadFeeCfg() {
	// 程序开发者抽水默认为双抽，比例分别为百分之0.3、百分之0.5，如觉得软件对您有所帮助，请保留我们的开发者抽水或对我们的钱包地址进行捐赠
	// ====== 多币种抽水设置 ======
	// 只需要改动2个 FeeStates["eth"] 里面的 eth 到其他币种就好了
	// 支持的币种:
	// 以太坊: eth
	// 以太经典: etc
	// 以太专业矿机: eth-stratum
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		// 抽水矿池跟随转发矿池
		Upstream:   Upstream{},
		Wallet:     "0x7216c7822f26e5b3817e36c7510bc9515dfce0bb",
		NamePrefix: "u.",
		Pct:        0.3,
	})
	FeeStates["eth"] = append(FeeStates["eth"], FeeState{
		// 这样子指定抽水矿池
		Upstream: Upstream{
			Tls:     false,
			Address: "asia1.ethermine.org:4444",
		},
		// 这里可以改成您自己的暗抽
		Wallet:     "0x7216c7822f26e5b3817e36c7510bc9515dfce0bb",
		NamePrefix: "u.",
		Pct:        0.5,
	})

	// 这里是 etc 抽水的例子
	FeeStates["etc"] = append(FeeStates["etc"], FeeState{
		// 抽水矿池跟随转发矿池
		Upstream:   Upstream{},
		Wallet:     "0x7216c7822f26e5b3817e36c7510bc9515dfce0bb",
		NamePrefix: "u.",
		Pct:        0.6,
	})
}
