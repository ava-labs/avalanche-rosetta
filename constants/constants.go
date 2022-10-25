package constants

import "errors"

var UnknownNetworkIdentifier = errors.New("unknown Network Identifier")

type NetworkIdentifiers uint16

const (
	// values are ASCII values because why not?
	Unknown NetworkIdentifiers = 0
	PChain  NetworkIdentifiers = 80 // "P"
	CChain  NetworkIdentifiers = 67 // "C"
	XChain  NetworkIdentifiers = 88 // "X"
)

func (ni NetworkIdentifiers) String() string {
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

func FromString(s string) (NetworkIdentifiers, error) {
	switch {
	case s == "P":
		return PChain, nil
	case s == "C":
		return CChain, nil
	case s == "X":
		return XChain, nil
	default:
		return Unknown, UnknownNetworkIdentifier
	}
}
