package burn

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: "xpla.burn.v1beta1.Query",
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "OngoingProposals",
					Use:       "ongoing-proposals",
					Short:     "Query all ongoing burn proposals",
				},
				{
					RpcMethod:      "OngoingProposal",
					Use:            "ongoing-proposal [proposal-id]",
					Short:          "Query a specific ongoing burn proposal by ID",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "proposal_id"}},
				},
			},
		},
	}
}
