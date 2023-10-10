<!-- This file is auto-generated. Please do not modify it yourself. -->
# Protobuf Documentation
<a name="top"></a>

## Table of Contents

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
  
- [Scalar Value Types](#scalar-value-types)



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



| Field | Type | Label | Description |
| ----- | ---- | ----- | ----------- |
| `title` | [string](#string) |  |  |
| `description` | [string](#string) |  |  |
| `validator_address` | [string](#string) |  |  |






<a name="xpla.volunteer.v1beta1.UnregisterVolunteerValidatorProposalWithDeposit"></a>

### UnregisterVolunteerValidatorProposalWithDeposit



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







<a name="xpla.volunteer.v1beta1.QueryVolunteerValidatorsResponse"></a>

### QueryVolunteerValidatorsResponse



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
| `VolunteerValidators` | [QueryVolunteerValidatorsRequest](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsRequest) | [QueryVolunteerValidatorsResponse](#xpla.volunteer.v1beta1.QueryVolunteerValidatorsResponse) |  | GET|/xpla/volun/v1beta1/validators|

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

