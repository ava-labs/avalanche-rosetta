package constants

type CChainOp uint16

const (
	Call CChainOp = iota + 1
	Fee
	Create
	Create2
	SelfDestruct

	CallCode
	DelegateCall
	StaticCall

	Destruct
	Import
	Export

	Erc20Transfer
	Erc20Mint
	Erc20Burn

	Erc721TransferSender
	Erc721TransferReceive
	Erc721Mint
	Erc721Burn
)

func (op CChainOp) String() string {
	switch op {
	case Call:
		return "CALL"
	case Fee:
		return "FEE"
	case Create:
		return "CREATE"
	case Create2:
		return "CREATE2"
	case SelfDestruct:
		return "SELFDESTRUCT"
	case CallCode:
		return "CALLCODE"
	case DelegateCall:
		return "DELEGATECALL"
	case StaticCall:
		return "STATICCALL"
	case Destruct:
		return "DESTRUCT"
	case Import:
		return "IMPORT"
	case Export:
		return "EXPORT"
	case Erc20Transfer:
		return "ERC20_TRANSFER"
	case Erc20Mint:
		return "ERC20_MINT"
	case Erc20Burn:
		return "ERC20_BURN"

	case Erc721TransferSender:
		return "ERC721_SENDER"
	case Erc721TransferReceive:
		return "ERC721_RECEIVE"
	case Erc721Mint:
		return "ERC721_MINT"
	case Erc721Burn:
		return "ERC721_BURN"

	default:
		return "" // TODO: FIND A DECENT DEFAULT VALUE
	}
}

var cOpsStrings = []string{
	Fee.String(),
	Call.String(),
	Create.String(),
	Create2.String(),
	SelfDestruct.String(),
	CallCode.String(),
	DelegateCall.String(),
	StaticCall.String(),
	Destruct.String(),
	Import.String(),
	Export.String(),
	Erc20Burn.String(),
	Erc20Mint.String(),
	Erc20Transfer.String(),
	Erc721TransferReceive.String(),
	Erc721TransferSender.String(),
	Erc721Mint.String(),
	Erc721Burn.String(),
}

func CChainOps() []string { return cOpsStrings }

var createTypes = []CChainOp{Create, Create2}

func IsCreation(t string) bool {
	for _, createType := range createTypes {
		if createType.String() == t {
			return true
		}
	}
	return false
}

var callTypes = []CChainOp{CallCode, DelegateCall, StaticCall}

func IsCall(t string) bool {
	for _, callType := range callTypes {
		if callType.String() == t {
			return true
		}
	}
	return false
}

func IsSelfDestruct(t string) bool {
	return SelfDestruct.String() == t
}

// IsAtomicOp determines whether a given C-chain operation is an atomic one
func IsAtomicOp(t string) bool {
	return t == Export.String() || t == Import.String()
}
