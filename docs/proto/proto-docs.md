<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

- [xpla/burn/v1beta1/burn.proto](#xpla/burn/v1beta1/burn.proto)
    - [BurnProposal](#xpla.burn.v1beta1.BurnProposal)
  
- [xpla/burn/v1beta1/genesis.proto](#xpla/burn/v1beta1/genesis.proto)
    - [GenesisState](#xpla.burn.v1beta1.GenesisState)
  
- [xpla/burn/v1beta1/query.proto](#xpla/burn/v1beta1/query.proto)
    - [QueryOngoingProposalRequest](#xpla.burn.v1beta1.QueryOngoingProposalRequest)
    - [QueryOngoingProposalResponse](#xpla.burn.v1beta1.QueryOngoingProposalResponse)
    - [QueryOngoingProposalsRequest](#xpla.burn.v1beta1.QueryOngoingProposalsRequest)
    - [QueryOngoingProposalsResponse](#xpla.burn.v1beta1.QueryOngoingProposalsResponse)
  
    - [Query](#xpla.burn.v1beta1.Query)
  
- [xpla/burn/v1beta1/tx.proto](#xpla/burn/v1beta1/tx.proto)
    - [MsgBurn](#xpla.burn.v1beta1.MsgBurn)
    - [MsgBurnResponse](#xpla.burn.v1beta1.MsgBurnResponse)
  
    - [Msg](#xpla.burn.v1beta1.Msg)
  
- [xpla/reward/v1beta1/reward.proto](#xpla/reward/v1beta1/reward.proto)
    - [Params](#xpla.reward.v1beta1.Params)
  
- [xpla/reward/v1beta1/genesis.proto](#xpla/reward/v1beta1/genesis.proto)
    - [GenesisState](#xpla.reward.v1beta1.GenesisState)
  
- [xpla/reward/v1beta1/query.proto](#xpla/reward/v1beta1/query.proto)
    - [QueryParamsRequest](#xpla.reward.v1beta1.QueryParamsRequest)
    - [QueryParamsResponse](#xpla.reward.v1beta1.QueryParamsResponse)
    - [QueryPoolRequest](#xpla.reward.v1beta1.QueryPoolRequest)
    - [QueryPoolResponse](#xpla.reward.v1beta1.QueryPoolResponse)
  
    - [Query](#xpla.reward.v1beta1.Query)
  
- [xpla/reward/v1beta1/tx.proto](#xpla/reward/v1beta1/tx.proto)
    - [MsgFundRewardPool](#xpla.reward.v1beta1.MsgFundRewardPool)
    - [MsgFundRewardPoolResponse](#xpla.reward.v1beta1.MsgFundRewardPoolResponse)
    - [MsgUpdateParams](#xpla.reward.v1beta1.MsgUpdateParams)
    - [MsgUpdateParamsResponse](#xpla.reward.v1beta1.MsgUpdateParamsResponse)
  
    - [Msg](#xpla.reward.v1beta1.Msg)
  
- [xpla/volunteer/v1beta1/volunteervalidator.proto](#xpla/volunteer/v1beta1/volunteervalidator.proto)
    - [VolunteerValidator](#xpla.volunteer.v1beta1.VolunteerValidator)
  
- [xpla/volunteer/v1beta1/genesis.proto](#xpla/volunteer/v1beta1/genesis.proto)
    - [GenesisState](#xpla.volunteer.v1beta1.GenesisState)
  
- [xpla/volunteer/v1beta1/proposal.proto](#xpla/volunteer/v1beta1/proposal.proto)
    - [RegisterVolunteerValidatorProposal](#xpla.volunteer.v1beta1.RegisterVolunteerValidatorProposal)
    - [RegisterVolunteerValidatorProposalWithDeposit](#xpla.volunteer.v1beta1.RegisterVolunteerValidatorProposalWithDeposit)
    - [UnregisterVolunteerValidatorProposal](#xpla.volunteer.v1beta1.UnregisterVolunteerValidatorProposal)
    - [UnregisterVolunteerValidatorProposalWithDeposit](#xpla.volunteer.v1beta1.UnregisterVolunteerValidatorProposalWithDeposit)
  
- [xpla/volunteer/v1beta1/query.proto](#xpla/volunteer/v1beta1/query.proto)
    - [QueryVolunteerValidatorsRequest](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsRequest)
    - [QueryVolunteerValidatorsResponse](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsResponse)
  
    - [Query](#xpla.volunteer.v1beta1.Query)
  
- [xpla/volunteer/v1beta1/tx.proto](#xpla/volunteer/v1beta1/tx.proto)
    - [MsgRegisterVolunteerValidator](#xpla.volunteer.v1beta1.MsgRegisterVolunteerValidator)
    - [MsgRegisterVolunteerValidatorResponse](#xpla.volunteer.v1beta1.MsgRegisterVolunteerValidatorResponse)
    - [MsgUnregisterVolunteerValidator](#xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidator)
    - [MsgUnregisterVolunteerValidatorResponse](#xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidatorResponse)
  
    - [Msg](#xpla.volunteer.v1beta1.Msg)
  
- [Scalar Value Types](#scalar-value-types)



<a name="xpla/burn/v1beta1/burn.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/burn/v1beta1/burn.proto



<a name="xpla.burn.v1beta1.BurnProposal"></a>

### BurnProposal
BurnProposal defines a ongoingburn proposal


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `proposal_id` | [uint64](#uint64) |  |  |
| `proposer` | [string](#string) |  |  |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/burn/v1beta1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/burn/v1beta1/genesis.proto



<a name="xpla.burn.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the bank module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `ongoing_burn_proposals` | [BurnProposal](#xpla.burn.v1beta1.BurnProposal) | repeated | ongoing_burn_proposals defines the ongoing burn proposals at genesis |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/burn/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/burn/v1beta1/query.proto



<a name="xpla.burn.v1beta1.QueryOngoingProposalRequest"></a>

### QueryOngoingProposalRequest
QueryOngoingProposalRequest is the request type for the Query/OngoingProposal
RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `proposal_id` | [uint64](#uint64) |  |  |






<a name="xpla.burn.v1beta1.QueryOngoingProposalResponse"></a>

### QueryOngoingProposalResponse
QueryOngoingProposalResponse is the response type for the
Query/OngoingProposal RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `proposer` | [string](#string) |  |  |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |






<a name="xpla.burn.v1beta1.QueryOngoingProposalsRequest"></a>

### QueryOngoingProposalsRequest
QueryOngoingProposalsRequest is the request type for the
Query/OngoingProposals RPC method.






<a name="xpla.burn.v1beta1.QueryOngoingProposalsResponse"></a>

### QueryOngoingProposalsResponse
QueryOngoingProposalsResponse is the response type for the
Query/OngoingProposals RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `proposals` | [BurnProposal](#xpla.burn.v1beta1.BurnProposal) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.burn.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service for burn module.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `OngoingProposals` | [QueryOngoingProposalsRequest](#xpla.burn.v1beta1.QueryOngoingProposalsRequest) | [QueryOngoingProposalsResponse](#xpla.burn.v1beta1.QueryOngoingProposalsResponse) | Query all ongoing burn proposals | GET|/xpla/burn/v1beta1/ongoing_proposals|
| `OngoingProposal` | [QueryOngoingProposalRequest](#xpla.burn.v1beta1.QueryOngoingProposalRequest) | [QueryOngoingProposalResponse](#xpla.burn.v1beta1.QueryOngoingProposalResponse) | Query a specific ongoing burn proposal by ID | GET|/xpla/burn/v1beta1/ongoing_proposal|

 <!-- end services -->



<a name="xpla/burn/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/burn/v1beta1/tx.proto



<a name="xpla.burn.v1beta1.MsgBurn"></a>

### MsgBurn
MsgBurn represents a message to burn coins from an account.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |






<a name="xpla.burn.v1beta1.MsgBurnResponse"></a>

### MsgBurnResponse
MsgBurnResponse defines the Msg/Burn response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.burn.v1beta1.Msg"></a>

### Msg
Msg defines the burn service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Burn` | [MsgBurn](#xpla.burn.v1beta1.MsgBurn) | [MsgBurnResponse](#xpla.burn.v1beta1.MsgBurnResponse) | Burn defines a method for burning coins from an account. | |

 <!-- end services -->



<a name="xpla/reward/v1beta1/reward.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/reward/v1beta1/reward.proto



<a name="xpla.reward.v1beta1.Params"></a>

### Params
Params defines the set of params for the reward module.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `fee_pool_rate` | [string](#string) |  |  |
| `community_pool_rate` | [string](#string) |  |  |
| `reserve_rate` | [string](#string) |  |  |
| `reserve_account` | [string](#string) |  |  |
| `reward_distribute_account` | [string](#string) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/reward/v1beta1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/reward/v1beta1/genesis.proto



<a name="xpla.reward.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the reward module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#xpla.reward.v1beta1.Params) |  | params defines all the paramaters of the module. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/reward/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/reward/v1beta1/query.proto



<a name="xpla.reward.v1beta1.QueryParamsRequest"></a>

### QueryParamsRequest
QueryParamsRequest is the request type for the Query/Params RPC method.






<a name="xpla.reward.v1beta1.QueryParamsResponse"></a>

### QueryParamsResponse
QueryParamsResponse is the response type for the Query/Params RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `params` | [Params](#xpla.reward.v1beta1.Params) |  | params defines the parameters of the module. |






<a name="xpla.reward.v1beta1.QueryPoolRequest"></a>

### QueryPoolRequest
QueryPoolRequest is the request type for the Query/Pool RPC
method.






<a name="xpla.reward.v1beta1.QueryPoolResponse"></a>

### QueryPoolResponse
QueryPoolResponse is the response type for the Query/Pool
RPC method.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `pool` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated | pool defines reward pool's coins. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.reward.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service for reward module.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `Params` | [QueryParamsRequest](#xpla.reward.v1beta1.QueryParamsRequest) | [QueryParamsResponse](#xpla.reward.v1beta1.QueryParamsResponse) | Params queries params of the reward module. | GET|/xpla/reward/v1beta1/params|
| `Pool` | [QueryPoolRequest](#xpla.reward.v1beta1.QueryPoolRequest) | [QueryPoolResponse](#xpla.reward.v1beta1.QueryPoolResponse) | Pool queries the reward module pool coins. | GET|/xpla/reward/v1beta1/pool|

 <!-- end services -->



<a name="xpla/reward/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/reward/v1beta1/tx.proto



<a name="xpla.reward.v1beta1.MsgFundRewardPool"></a>

### MsgFundRewardPool
MsgFundRewardPool allows an account to directly
fund the reward pool.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |
| `depositor` | [string](#string) |  |  |






<a name="xpla.reward.v1beta1.MsgFundRewardPoolResponse"></a>

### MsgFundRewardPoolResponse
MsgFundRewardPoolResponse defines the Msg/FundRewardPool response type.






<a name="xpla.reward.v1beta1.MsgUpdateParams"></a>

### MsgUpdateParams
MsgUpdateParams is the Msg/UpdateParams request type for reward parameters.
Since: cosmos-sdk 0.47


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `params` | [Params](#xpla.reward.v1beta1.Params) |  | params defines the x/evm parameters to update. NOTE: All parameters must be supplied. |






<a name="xpla.reward.v1beta1.MsgUpdateParamsResponse"></a>

### MsgUpdateParamsResponse
MsgUpdateParamsResponse defines the response structure for executing a
MsgUpdateParams message.
Since: cosmos-sdk 0.47





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.reward.v1beta1.Msg"></a>

### Msg
Msg defines the reawrd Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `FundRewardPool` | [MsgFundRewardPool](#xpla.reward.v1beta1.MsgFundRewardPool) | [MsgFundRewardPoolResponse](#xpla.reward.v1beta1.MsgFundRewardPoolResponse) | MsgFundRewardPool defines a method to allow an account to directly fund the reward pool. | |
| `UpdateParams` | [MsgUpdateParams](#xpla.reward.v1beta1.MsgUpdateParams) | [MsgUpdateParamsResponse](#xpla.reward.v1beta1.MsgUpdateParamsResponse) | UpdateParams defined a governance operation for updating the x/reward module parameters. The authority is hard-coded to the Cosmos SDK x/gov module account | |

 <!-- end services -->



<a name="xpla/volunteer/v1beta1/volunteervalidator.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/volunteer/v1beta1/volunteervalidator.proto



<a name="xpla.volunteer.v1beta1.VolunteerValidator"></a>

### VolunteerValidator
VolunteerValidator required for validator set update logic.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `address` | [string](#string) |  | address is the address of the validator. |
| `power` | [int64](#int64) |  | power defines the power of the validator. |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/volunteer/v1beta1/genesis.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/volunteer/v1beta1/genesis.proto



<a name="xpla.volunteer.v1beta1.GenesisState"></a>

### GenesisState
GenesisState defines the volunteer module's genesis state.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `volunteer_validators` | [VolunteerValidator](#xpla.volunteer.v1beta1.VolunteerValidator) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/volunteer/v1beta1/proposal.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/volunteer/v1beta1/proposal.proto



<a name="xpla.volunteer.v1beta1.RegisterVolunteerValidatorProposal"></a>

### RegisterVolunteerValidatorProposal
RegisterVolunteerValidatorProposal


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  |  |
| `description` | [string](#string) |  |  |
| `validator_description` | [cosmos.staking.v1beta1.Description](#cosmos.staking.v1beta1.Description) |  |  |
| `delegator_address` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |
| `pubkey` | [google.protobuf.Any](#google.protobuf.Any) |  |  |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |  |  |






<a name="xpla.volunteer.v1beta1.RegisterVolunteerValidatorProposalWithDeposit"></a>

### RegisterVolunteerValidatorProposalWithDeposit
RegisterVolunteerValidatorProposalWithDeposit


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  |  |
| `description` | [string](#string) |  |  |
| `validator_description` | [cosmos.staking.v1beta1.Description](#cosmos.staking.v1beta1.Description) |  |  |
| `delegator_address` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |
| `pubkey` | [google.protobuf.Any](#google.protobuf.Any) |  |  |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |  |  |
| `deposit` | [string](#string) |  |  |






<a name="xpla.volunteer.v1beta1.UnregisterVolunteerValidatorProposal"></a>

### UnregisterVolunteerValidatorProposal
UnregisterVolunteerValidatorProposal


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  |  |
| `description` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |






<a name="xpla.volunteer.v1beta1.UnregisterVolunteerValidatorProposalWithDeposit"></a>

### UnregisterVolunteerValidatorProposalWithDeposit
UnregisterVolunteerValidatorProposalWithDeposit


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  |  |
| `description` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |
| `deposit` | [string](#string) |  |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->

 <!-- end services -->



<a name="xpla/volunteer/v1beta1/query.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/volunteer/v1beta1/query.proto



<a name="xpla.volunteer.v1beta1.QueryVolunteerValidatorsRequest"></a>

### QueryVolunteerValidatorsRequest
QueryVolunteerValidatorsRequest






<a name="xpla.volunteer.v1beta1.QueryVolunteerValidatorsResponse"></a>

### QueryVolunteerValidatorsResponse
QueryVolunteerValidatorsResponse


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `volunteer_validators` | [string](#string) | repeated |  |





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.volunteer.v1beta1.Query"></a>

### Query
Query defines the gRPC querier service for volunteer module.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `VolunteerValidators` | [QueryVolunteerValidatorsRequest](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsRequest) | [QueryVolunteerValidatorsResponse](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsResponse) | VolunteerValidators | GET|/xpla/volunteer/v1beta1/validators|

 <!-- end services -->



<a name="xpla/volunteer/v1beta1/tx.proto"></a>
<p align="right"><a href="#top">Top</a></p>

## xpla/volunteer/v1beta1/tx.proto



<a name="xpla.volunteer.v1beta1.MsgRegisterVolunteerValidator"></a>

### MsgRegisterVolunteerValidator
MsgRegisterVolunteerValidator defines a message to register a new volunteer
validator.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `validator_description` | [cosmos.staking.v1beta1.Description](#cosmos.staking.v1beta1.Description) |  |  |
| `delegator_address` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |
| `pubkey` | [google.protobuf.Any](#google.protobuf.Any) |  |  |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) |  |  |






<a name="xpla.volunteer.v1beta1.MsgRegisterVolunteerValidatorResponse"></a>

### MsgRegisterVolunteerValidatorResponse
MsgRegisterVolunteerValidatorResponse defines the RegisterVolunteerValidator
response.






<a name="xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidator"></a>

### MsgUnregisterVolunteerValidator
MsgUnregisterVolunteerValidator defines a message to unregister a volunteer
validator.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `authority` | [string](#string) |  | authority is the address of the governance account. |
| `validator_address` | [string](#string) |  |  |






<a name="xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidatorResponse"></a>

### MsgUnregisterVolunteerValidatorResponse
MsgUnregisterVolunteerValidatorResponse defines the
UnregisterVolunteerValidator response.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.volunteer.v1beta1.Msg"></a>

### Msg
Msg defines the volunteer Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `RegisterVolunteerValidator` | [MsgRegisterVolunteerValidator](#xpla.volunteer.v1beta1.MsgRegisterVolunteerValidator) | [MsgRegisterVolunteerValidatorResponse](#xpla.volunteer.v1beta1.MsgRegisterVolunteerValidatorResponse) | RegisterVolunteerValidator defines a method to register a new volunteer validator. | |
| `UnregisterVolunteerValidator` | [MsgUnregisterVolunteerValidator](#xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidator) | [MsgUnregisterVolunteerValidatorResponse](#xpla.volunteer.v1beta1.MsgUnregisterVolunteerValidatorResponse) | UnregisterVolunteerValidator defines a method to unregister a volunteer | |

 <!-- end services -->



## Scalar Value Types

| .proto Type | Notes | C++ | Java | Python | Go | C# | PHP | Ruby |
| ----------- | ----- | --- | ---- | ------ | -- | -- | --- | ---- |
| <a name="double" /> double |  | double | double | float | float64 | double | float | Float |
| <a name="float" /> float |  | float | float | float | float32 | float | float | Float |
| <a name="int32" /> int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="int64" /> int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="uint32" /> uint32 | Uses variable-length encoding. | uint32 | int | int/long | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="uint64" /> uint64 | Uses variable-length encoding. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum or Fixnum (as required) |
| <a name="sint32" /> sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sint64" /> sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="fixed32" /> fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int | uint32 | uint | integer | Bignum or Fixnum (as required) |
| <a name="fixed64" /> fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long | uint64 | ulong | integer/string | Bignum |
| <a name="sfixed32" /> sfixed32 | Always four bytes. | int32 | int | int | int32 | int | integer | Bignum or Fixnum (as required) |
| <a name="sfixed64" /> sfixed64 | Always eight bytes. | int64 | long | int/long | int64 | long | integer/string | Bignum |
| <a name="bool" /> bool |  | bool | boolean | boolean | bool | bool | boolean | TrueClass/FalseClass |
| <a name="string" /> string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode | string | string | string | String (UTF-8) |
| <a name="bytes" /> bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str | []byte | ByteString | string | String (ASCII-8BIT) |

