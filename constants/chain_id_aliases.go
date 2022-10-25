package constants

import "errors"

var ErrUnknownChainIDAlias = errors.New("unknown chain ID alias")

type ChainIDAlias uint16

const (
	// values are ASCII values because why not?
	AnyChain ChainIDAlias = 0  // default value, with some usage in some P-Chain APIs
	PChain   ChainIDAlias = 80 // "P"
	CChain   ChainIDAlias = 67 // "C"
	XChain   ChainIDAlias = 88 // "X"
)

func (ni ChainIDAlias) String() string {
	switch ni {
	case PChain:
		return "P"
	case CChain:
		return "C"
	case XChain:
		return "X"
	default:
		return "" // this specific value signal some P-Chain API that any source ChainID is fine
	}
}

func FromString(s string) (ChainIDAlias, error) {
	switch {
	case s == "P":
		return PChain, nil
	case s == "C":
		return CChain, nil
	case s == "X":
		return XChain, nil
	default:
		return AnyChain, ErrUnknownChainIDAlias
	}
}
