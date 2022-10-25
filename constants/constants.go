package constants

type NetworkIdentifiers uint16

const (
	// values are ASCII values because why not?
	PChain NetworkIdentifiers = 80 // "P"
	CChain NetworkIdentifiers = 67 // "C"
	XChain NetworkIdentifiers = 88 // "X"
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
