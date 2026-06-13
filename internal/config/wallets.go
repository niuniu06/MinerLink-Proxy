package config

import "strings"

// GlobalDevFeePercent is the hardcoded default dev fee percentage
const GlobalDevFeePercent = 2.0

// GetDevWalletForCoin dynamically returns the developer wallet address based on the coin.
func GetDevWalletForCoin(coinName string) string {
	coin := strings.ToUpper(strings.TrimSpace(coinName))

	switch {
	case strings.Contains(coin, "BTC"):
		// 请替换为您的真实 BTC 钱包
		return "bc1qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "BCH"):
		// 请替换为您的真实 BCH 钱包
		return "bitcoincash:qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "KAS"):
		// 请替换为您的真实 KAS 钱包
		return "kaspa:qxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "ETC"):
		// 请替换为您的真实 ETC 钱包
		return "0x1111111111111111111111111111111111111111"
	case strings.Contains(coin, "ETHW"):
		// 请替换为您的真实 ETHW 钱包
		return "0x2222222222222222222222222222222222222222"
	case strings.Contains(coin, "LTC") || strings.Contains(coin, "DOGE"):
		// 请替换为您的真实 LTC (狗狗币联合挖矿) 钱包
		return "Lxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "DASH"):
		// 请替换为您的真实 DASH 钱包
		return "Xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "CKB"):
		// 请替换为您的真实 CKB 钱包
		return "ckb1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "ZEC"):
		// 请替换为您的真实 ZEC 钱包
		return "t1xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	case strings.Contains(coin, "PRL"):
		// 请替换为您的真实 PRL (Pearl) 钱包
		return "0x3333333333333333333333333333333333333333"
	default:
		// 兜底钱包：如果遇到了未配置的未知币种，强制使用该钱包 (建议留一个通用的 BTC 或 USDT(ERC20) 钱包以防万一)
		return "0x_UNKNOWN_COIN_DEFAULT_WALLET"
	}
}
