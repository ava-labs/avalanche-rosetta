package pchain

type AccountBalance struct {
	Total              uint64
	Unlocked           uint64
	Staked             uint64
	LockedStakeable    uint64
	LockedNotStakeable uint64
}
