package kernel

import "errors"

type TokenBillingTerm struct {
	Tokens            int64
	PriceMicrocredits int64
}

// TokenBillingAmount 按百万 Token 单价累加，倍率 10_000 表示 1.00 倍。
func TokenBillingAmount(multiplierBPS int64, terms ...TokenBillingTerm) (int64, error) {
	if multiplierBPS <= 0 {
		return 0, errors.New("积分计费参数无效")
	}
	var total int64
	for _, term := range terms {
		if term.Tokens <= 0 || term.PriceMicrocredits == 0 {
			continue
		}
		if term.PriceMicrocredits < 0 {
			return 0, errors.New("积分计费参数无效")
		}
		if term.PriceMicrocredits > (1<<63-1)/term.Tokens {
			return 0, errors.New("积分计费金额溢出")
		}
		base := term.PriceMicrocredits * term.Tokens
		if base > ((1<<63-1)-999_999)/1 {
			return 0, errors.New("积分计费金额溢出")
		}
		// 单价是每百万 Token 的微积分。
		chunk := (base + 999_999) / 1_000_000
		if chunk > ((1<<63-1)-9_999)/multiplierBPS {
			return 0, errors.New("积分计费金额溢出")
		}
		amount := (chunk*multiplierBPS + 9_999) / 10_000
		if total > (1<<63-1)-amount {
			return 0, errors.New("积分计费金额溢出")
		}
		total += amount
	}
	return total, nil
}
