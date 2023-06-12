<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

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
    - [MsgFundFeeCollector](#xpla.reward.v1beta1.MsgFundFeeCollector)
    - [MsgFundFeeCollectorResponse](#xpla.reward.v1beta1.MsgFundFeeCollectorResponse)
  
    - [Msg](#xpla.reward.v1beta1.Msg)
  
- [Scalar Value Types](#scalar-value-types)



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



<a name="xpla.reward.v1beta1.MsgFundFeeCollector"></a>

### MsgFundFeeCollector
MsgFundFeeCollector allows an account to directly
fund the fee collector.


| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `amount` | [cosmos.base.v1beta1.Coin](#cosmos.base.v1beta1.Coin) | repeated |  |
| `depositor` | [string](#string) |  |  |






<a name="xpla.reward.v1beta1.MsgFundFeeCollectorResponse"></a>

### MsgFundFeeCollectorResponse
MsgFundFeeCollectorResponse defines the Msg/FundFeeCollector response type.





 <!-- end messages -->

 <!-- end enums -->

 <!-- end HasExtensions -->


<a name="xpla.reward.v1beta1.Msg"></a>

### Msg
Msg defines the reawrd Msg service.

| Method Name | Request Type | Response Type | Description | HTTP Verb | Endpoint |
| ----------- | ------------ | ------------- | ------------| ------- | -------- |
| `FundFeeCollector` | [MsgFundFeeCollector](#xpla.reward.v1beta1.MsgFundFeeCollector) | [MsgFundFeeCollectorResponse](#xpla.reward.v1beta1.MsgFundFeeCollectorResponse) | FundFeeCollector defines a method to allow an account to directly fund the fee collector. | |

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

