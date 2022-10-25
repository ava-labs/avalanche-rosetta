package constants

type PChainTxType uint16

const (
	ImportAvax PChainTxType = iota + 1
	ExportAvax
	AddValidator
	AddPermissionlessValidator
	AddDelegator
	AddPermissionlessDelegator
	RewardValidator
	CreateChain
	CreateSubnet
	AddSubnetValidator
	RemoveSubnetValidator
	TransformSubnetValidator
	AdvanceTime
)

func (op PChainTxType) String() string {
	switch op {
	case ImportAvax:
		return "IMPORT_AVAX"
	case ExportAvax:
		return "EXPORT_AVAX"
	case AddValidator:
		return "ADD_VALIDATOR"
	case AddPermissionlessValidator:
		return "ADD_PERMISSIONLESS_VALIDATOR"
	case AddDelegator:
		return "ADD_DELEGATOR"
	case AddPermissionlessDelegator:
		return "ADD_PERMISSIONLESS_DELEGATOR"
	case RewardValidator:
		return "REWARD_VALIDATOR"
	case CreateChain:
		return "CREATE_CHAIN"
	case CreateSubnet:
		return "CREATE_SUBNET"
	case AddSubnetValidator:
		return "ADD_SUBNET_VALIDATOR"
	case RemoveSubnetValidator:
		return "REMOVE_SUBNET_VALIDATOR"
	case TransformSubnetValidator:
		return "TRANSFORM_SUBNET_VALIDATOR"
	case AdvanceTime:
		return "ADVANCE_TIME"

	default:
		return "" // TODO: FIND A DECENT DEFAULT VALUE
	}
}

var pTxTypesStrings = []string{
	ImportAvax.String(),
	ExportAvax.String(),
	AddValidator.String(),
	AddDelegator.String(),
	RewardValidator.String(),
	CreateChain.String(),
	CreateSubnet.String(),
	AddSubnetValidator.String(),
	RemoveSubnetValidator.String(),
	TransformSubnetValidator.String(),
	AddPermissionlessValidator.String(),
	AddPermissionlessDelegator.String(),
}

func PChainTxTypes() []string { return pTxTypesStrings }
