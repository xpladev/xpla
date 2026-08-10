# CONX Dynamic Deflation Operations

This document describes the operator-visible behavior of the `x/dynamicdeflation`
module. All amounts in module state, queries, and events use the integer on-chain
denomination `axpla` (`1 XPLA = 10^18 axpla`).

## Block Processing

The application runs the relevant BeginBlock modules in this order:

```text
mint -> dynamicdeflation -> distribution
```

At each observed block, `dynamicdeflation` reads the complete `axpla` balance of
the FeeCollector before `distribution` runs. This balance is the block's
`gross` amount. It is not a transaction-fee counter. It can include:

- transaction fees left from the previous block;
- the current block's mint inflow;
- `x/reward` funds sent to FeeCollector at the end of the previous block; and
- any other `axpla` held by FeeCollector at that point.

Funds that do not pass through FeeCollector, including direct `x/reward`
transfers to the Community Pool or Reserve, are not part of `gross`.

The module routes only `axpla`:

```text
allocated = floor(gross * allocation_rate)
distribution_remainder = gross - allocated
```

It transfers `allocated` from FeeCollector to the `dynamicdeflation` module
account and leaves `distribution_remainder` in FeeCollector for the standard
distribution module. It does not move, track, burn, or fund the Community Pool
with any other denomination. If `allocated` is zero, it does not perform a bank
transfer.

The module account is named `dynamicdeflation`, has the `Burner` permission,
and remains covered by the application's blocked module-address policy. Its
balance must not be treated as a general-purpose deposit account.

The following conservation rule applies to every observed block:

```text
allocated + distribution_remainder = gross
```

`x/reward` runs after distribution. An `axpla` amount that it sends to
FeeCollector in block `B` is therefore included once in the dynamic-deflation
`gross` observed in block `B+1`.

## Periods and Settlement

Settlement heights are globally aligned to `settlement_interval_blocks`. For
interval `N`, routing happens every observed block and settlement happens after
routing whenever:

```text
height % N = 0
```

For example, interval `100000` settles at heights `100000` and `5200000`. If the
first observed height `H` is not aligned, the first period ends at the next
multiple of `N` and is shorter than `N` blocks. If `H` is already aligned, that
block is routed and settled immediately. Subsequent periods begin on the next
block and contain exactly `N` observed blocks.

The module always routes and accumulates the settlement block before settling.
A 100,000-block period is not a calendar day, month, or any other fixed wall-clock
duration; its elapsed time varies with block production.

At genesis height 1, the module performs no routing and creates no period, which
matches distribution's FeeCollector handling at that height. A fresh chain's
first period starts at height 2. For a mainnet upgrade at height `H > 1`, the
upgrade block is included as the first observed block and the first settlement
occurs at the first multiple of the active interval at or after `H`.

For each period, let:

- `F` be `gross_amount`, the sum of all observed FeeCollector `gross` amounts;
- `T` be `allocated_amount`, the sum routed to the dynamicdeflation module account;
- `Min` be `min_fee_amount`; and
- `Max` be `max_fee_amount`.

Settlement uses the active period configuration:

```text
F <= Min: community = 0; burn = T
F >= Max: community = T; burn = 0
Min < F < Max:
  community = floor(T * (F - Min) / (Max - Min))
  burn = T - community
```

The implementation uses deterministic decimal truncation and integer
multiplication followed by integer division. It does not use floating-point
arithmetic. Calculating `burn` as the remainder preserves:

```text
burn + community = T
```

The community amount is sent through the existing distribution Community Pool
using `FundCommunityPool`. The burn amount is destroyed immediately from the
`dynamicdeflation` module account with `BurnCoins`. There is no Burn Pending
account, burn proposal, or later governance approval. **Automatic burn is
irreversible once the block is committed.** Existing Community Pool balances
are never included in this calculation or moved by settlement.

## Parameters and Governance

The default parameter values are:

| Parameter | Default |
| --- | --- |
| `enabled` | `true` |
| `allocation_rate` | `0.20` |
| `settlement_interval_blocks` | `100000` |
| `min_fee_amount` | `10000 XPLA` (`10000 * 10^18 axpla`) |
| `max_fee_amount` | `500000 XPLA` (`500000 * 10^18 axpla`) |

Governance updates all fields atomically through `MsgUpdateParams`. `Params`
are candidate settings for the next period. When a period starts, the module
copies the complete current `Params` into `CurrentPeriod.active_config`.
Changing the rate, interval, thresholds, or enabled state does not modify an
active period, its end height, or its settlement formula. If governance updates
parameters more than once during a period, the values present when the next
period starts take effect.

Only the governance module address can authorize `MsgUpdateParams`. The module
rejects the complete update without storing any field when the rate is outside
`[0, 1]`, the interval is less than 1, either threshold is malformed or
negative, the threshold denomination is not `axpla`, or `max_fee_amount` is not
greater than `min_fee_amount`. Nil or empty decimal rates are also invalid.

Disabling the module does not act as an emergency pause:

1. An active period continues using its snapshot and settles normally.
2. After settlement, the module does not create another period or route funds
   while the latest `Params.enabled` is `false`.
3. During that time, standard distribution processes the complete FeeCollector
   balance.
4. After governance re-enables the module, the following BeginBlock starts a
   complete new period using the latest parameters.

The v1.12 binary registers its activation handler under the on-chain upgrade
name `v1_12`. That handler adds the `dynamicdeflation` store, initializes
the module through `RunMigrations`, and sets the existing distribution parameter
`community_tax` to zero while preserving the other distribution parameters.
It also configures the on-chain fee market so a transaction with a 100,000 gas
limit has a minimum fee of 1 XPLA:

```text
1 XPLA / 100,000 gas = 10,000,000,000,000 axpla/gas
```

At the upgrade height, both `min_gas_price` and `base_fee` are set to this value,
base-fee operation is enabled, and `enable_height` is set to the upgrade height.
Other fee-market parameters are preserved. EIP-1559 may raise the base fee on
later blocks, while `min_gas_price` prevents it from falling below this floor.
This is not a permanent restriction on distribution governance. If governance
later raises `community_tax`, standard distribution applies that tax to the
FeeCollector remainder after dynamic routing. Operators must treat such a
change as an additional tokenomics change because it adds Community Pool
allocation on top of dynamic-deflation settlement.

The upgrade height remains a deployment choice and is not inferred from module
defaults.

## Queries

All amount fields below are exact `axpla` integers, not display-denominated XPLA
values.

| Query | Operational use | Important fields |
| --- | --- | --- |
| `Params` | Inspect the candidate configuration for the next period. | `enabled`, `allocation_rate`, `settlement_interval_blocks`, `min_fee_amount`, `max_fee_amount` |
| `CurrentPeriod` | Inspect the active snapshot and accumulated amounts. | `start_height`, `end_height`, `active_config`, `gross_amount`, `allocated_amount` |
| `Status` | Compare the module account balance with the active period allocation. | `module_balance`, `allocated_amount`, `surplus_amount`, `deficit_amount` |

`Status.allocated_amount` is the active period's allocated amount.
`module_balance` is the module account's actual `axpla` balance. A balance above
the allocated amount appears as `surplus_amount`; a balance below it appears as
`deficit_amount`. Other denominations are not part of these settlement amounts.

A surplus is observable but is not automatically burned or transferred to the
Community Pool. A deficit is an invariant violation: the module does not settle
only the available balance or partially settle the period.

## Events and Verification

Routing emits `dynamic_deflation_routed` with:

- `height`
- `gross`
- `allocated`
- `remainder`

Settlement emits `dynamic_deflation_settled` with:

- `start_height`
- `end_height`
- `settlement_height`
- `gross`
- `allocated`
- `burn`
- `community`

BeginBlock processing has no transaction and therefore no transaction hash.
The module does not persist completed periods as application state. The
`dynamic_deflation_settled` block event is the historical settlement record, so
operators that require long-term history must retain or index block events.
Use the following evidence to verify a routing or settlement operation:

1. Identify the block height and event.
2. Use the event's start, end, and settlement heights as the settlement
   identifiers.
3. Confirm `allocated + remainder = gross` for a routing event.
4. Confirm `burn + community = allocated` for a settlement event.
5. Confirm the burn against the total-supply decrease and the community amount
   against the existing Community Pool accounting change.
6. Inspect `Status` for any surplus or deficit.

Do not search for a settlement transaction hash; block heights and BeginBlock
events are the audit keys.

## Failure and Invariant Handling

Routing and settlement execute in one cache context. The module commits state
and events only after the full operation succeeds. A FeeCollector transfer,
Community Pool funding, burn, or state-validation error rolls back all dynamic
deflation changes and returns a BeginBlock error. The application does not
ignore the error and continue to distribution.

Settlement operates on `CurrentPeriod.allocated_amount`, not the module
account's complete balance. Before settlement, the module requires the actual
`axpla` balance to cover that allocated amount:

- If the balance is lower, settlement fails closed without a partial burn or
  Community Pool transfer.
- If the balance is higher, only the allocated amount is settled and the surplus
  remains in the module account.
- Other denominations in the module account are never settled.

Operators should alert on any non-zero `deficit_amount`. A non-zero
`surplus_amount` also requires investigation, even though it does not block
settlement of the allocated amount.
