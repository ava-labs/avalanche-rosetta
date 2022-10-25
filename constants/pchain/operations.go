package pconstants

type Op uint16

const (
	Import Op = iota + 1
	Export
	Input
	Output
	Stake
	Reward
)

func (op Op) String() string {
	switch op {
	case Import:
		return "IMPORT"
	case Export:
		return "EXPORT"
	case Input:
		return "INPUT"
	case Output:
		return "OUTPUT"
	case Stake:
		return "STAKE"
	case Reward:
		return "REWARD"

	default:
		return "" // TODO: FIND A DECENT DEFAULT VALUE
	}
}
