package types

type MethodErc20 string

const (
	Allowance    MethodErc20 = "allowance"
	Approve      MethodErc20 = "approve"
	BalanceOf    MethodErc20 = "balanceOf"
	TotalSupply  MethodErc20 = "totalSupply"
	Transfer     MethodErc20 = "transfer"
	TransferFrom MethodErc20 = "transferFrom"
)

func GetErc20Method(name MethodErc20) string {
	return string(name)
}
