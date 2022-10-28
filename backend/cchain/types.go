package cchain

const BalanceOfMethodPrefix = "0x70a08231000000000000000000000000"

type accountMetadata struct {
	Nonce uint64 `json:"nonce"`
}

// has0xPrefix validates str begins with '0x' or '0X'.
// Copied from the go-ethereum hextuil.go library
func has0xPrefix(str string) bool {
	return len(str) >= 2 && str[0] == '0' && (str[1] == 'x' || str[1] == 'X')
}
