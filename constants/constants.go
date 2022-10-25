package constants

import "errors"

var ErrUnknownChainIDAlias = errors.New("unknown chain ID alias")

type ChainIDAlias uint16

const (
	// values are ASCII values because why not?
	Unknown ChainIDAlias = 0
	PChain  ChainIDAlias = 80 // "P"
	CChain  ChainIDAlias = 67 // "C"
	XChain  ChainIDAlias = 88 // "X"
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
		return "Unknow"
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
		return Unknown, ErrUnknownChainIDAlias
	}
}
