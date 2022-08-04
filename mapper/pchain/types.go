package pchain

const (
	OpImportAvax         = "IMPORT_AVAX"
	OpExportAvax         = "EXPORT_AVAX"
	OpAddValidator       = "ADD_VALIDATOR"
	OpAddDelegator       = "ADD_DELEGATOR"
	OpRewardValidator    = "REWARD_VALIDATOR"
	OpCreateChain        = "CREATE_CHAIN"
	OpCreateSubnet       = "CREATE_SUBNET"
	OpAddSubnetValidator = "ADD_SUBNET_VALIDATOR"
)

var (
	OperationTypes = []string{
		OpImportAvax,
		OpExportAvax,
		OpAddValidator,
		OpAddDelegator,
		OpRewardValidator,
		OpCreateChain,
		OpCreateSubnet,
		OpAddSubnetValidator,
	}
	CallMethods = []string{}
)
