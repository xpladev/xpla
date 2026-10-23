package dynamicdeflation

import autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

func (AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: "xpla.dynamicdeflation.v1beta1.Query",
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{RpcMethod: "Params", Use: "params", Short: "Query dynamic deflation parameters"},
				{RpcMethod: "CurrentPeriod", Use: "current-period", Short: "Query the active dynamic deflation period"},
				{RpcMethod: "Status", Use: "status", Short: "Query the Dynamic Deflation Pool account status"},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service: "xpla.dynamicdeflation.v1beta1.Msg",
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{RpcMethod: "UpdateParams", Skip: true},
			},
		},
	}
}
