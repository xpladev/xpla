use cosmwasm_std::{
    entry_point, to_json_binary, BankMsg, Binary, Coin, Deps, DepsMut, Env, MessageInfo, Reply,
    Response, StdError, StdResult, Storage, SubMsg, SubMsgResult, Uint128, WasmMsg,
};
use serde::{Deserialize, Serialize};

const OBSERVE_REPLY_ID: u64 = 1;
const CHILD_FAILURE_REPLY_ID: u64 = 2;
const HANDLED_SEND_REPLY_ID: u64 = 3;

const OBSERVE_ADDRESS_KEY: &[u8] = b"observe_address";
const OBSERVE_DENOM_KEY: &[u8] = b"observe_denom";
const OBSERVED_AMOUNT_KEY: &[u8] = b"observed_amount";
const REPLY_CAUGHT_KEY: &[u8] = b"reply_caught";
const REPLY_ERROR_KEY: &[u8] = b"reply_error";

#[derive(Serialize, Deserialize)]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    SendThenObserve {
        denom: String,
        recipient: String,
        amount: Uint128,
    },
    SiblingThenHandledFailure {
        denom: String,
        earlier_recipient: String,
        earlier_amount: Uint128,
        child_contract: String,
        child_recipient: String,
        child_amount: Uint128,
        native_denom: String,
        native_amount: Uint128,
    },
    ChildTransferThenFail {
        denom: String,
        recipient: String,
        amount: Uint128,
        native_denom: String,
        native_amount: Uint128,
    },
    HandledSend {
        denom: String,
        recipient: String,
        amount: Uint128,
    },
    NativeThenXerc20 {
        native_denom: String,
        native_recipient: String,
        native_amount: Uint128,
        xerc20_denom: String,
        xerc20_recipient: String,
        xerc20_amount: Uint128,
    },
    NativeSiblingThenHandledFailure {
        native_denom: String,
        native_recipient: String,
        native_amount: Uint128,
        child_contract: String,
        child_recipient: String,
        child_amount: Uint128,
        failing_native_amount: Uint128,
        xerc20_denom: String,
    },
}

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    Observation {},
    ReplyState {},
}

#[derive(Serialize)]
pub struct ObservationResponse {
    pub amount: Uint128,
}

#[derive(Serialize)]
pub struct ReplyStateResponse {
    pub caught: bool,
    pub error: String,
}

#[entry_point]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    reset_reply_state(deps.storage);
    deps.storage
        .set(OBSERVED_AMOUNT_KEY, Uint128::zero().to_string().as_bytes());
    Ok(Response::new().add_attribute("contract", "xerc20-atomicity"))
}

#[entry_point]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::SendThenObserve {
            denom,
            recipient,
            amount,
        } => {
            deps.api.addr_validate(&recipient)?;
            deps.storage.set(OBSERVE_ADDRESS_KEY, recipient.as_bytes());
            deps.storage.set(OBSERVE_DENOM_KEY, denom.as_bytes());
            deps.storage
                .set(OBSERVED_AMOUNT_KEY, Uint128::zero().to_string().as_bytes());

            Ok(Response::new()
                .add_submessage(SubMsg::reply_on_success(
                    BankMsg::Send {
                        to_address: recipient,
                        amount: vec![Coin { denom, amount }],
                    },
                    OBSERVE_REPLY_ID,
                ))
                .add_attribute("action", "send_then_observe"))
        }
        ExecuteMsg::SiblingThenHandledFailure {
            denom,
            earlier_recipient,
            earlier_amount,
            child_contract,
            child_recipient,
            child_amount,
            native_denom,
            native_amount,
        } => {
            deps.api.addr_validate(&earlier_recipient)?;
            deps.api.addr_validate(&child_contract)?;
            deps.api.addr_validate(&child_recipient)?;
            reset_reply_state(deps.storage);

            let child_msg = to_json_binary(&ExecuteMsg::ChildTransferThenFail {
                denom: denom.clone(),
                recipient: child_recipient,
                amount: child_amount,
                native_denom,
                native_amount,
            })?;

            Ok(Response::new()
                .add_message(BankMsg::Send {
                    to_address: earlier_recipient,
                    amount: vec![Coin {
                        denom,
                        amount: earlier_amount,
                    }],
                })
                .add_submessage(SubMsg::reply_on_error(
                    WasmMsg::Execute {
                        contract_addr: child_contract,
                        msg: child_msg,
                        funds: vec![],
                    },
                    CHILD_FAILURE_REPLY_ID,
                ))
                .add_attribute("action", "sibling_then_handled_failure"))
        }
        ExecuteMsg::ChildTransferThenFail {
            denom,
            recipient,
            amount,
            native_denom,
            native_amount,
        } => {
            deps.api.addr_validate(&recipient)?;
            Ok(Response::new()
                .add_message(BankMsg::Send {
                    to_address: recipient.clone(),
                    amount: vec![Coin { denom, amount }],
                })
                .add_message(BankMsg::Send {
                    to_address: recipient,
                    amount: vec![Coin {
                        denom: native_denom,
                        amount: native_amount,
                    }],
                })
                .add_attribute("action", "child_transfer_then_fail"))
        }
        ExecuteMsg::HandledSend {
            denom,
            recipient,
            amount,
        } => {
            deps.api.addr_validate(&recipient)?;
            reset_reply_state(deps.storage);
            Ok(Response::new()
                .add_submessage(SubMsg::reply_on_error(
                    BankMsg::Send {
                        to_address: recipient,
                        amount: vec![Coin { denom, amount }],
                    },
                    HANDLED_SEND_REPLY_ID,
                ))
                .add_attribute("action", "handled_send"))
        }
        ExecuteMsg::NativeThenXerc20 {
            native_denom,
            native_recipient,
            native_amount,
            xerc20_denom,
            xerc20_recipient,
            xerc20_amount,
        } => {
            deps.api.addr_validate(&native_recipient)?;
            deps.api.addr_validate(&xerc20_recipient)?;
            Ok(Response::new()
                .add_message(BankMsg::Send {
                    to_address: native_recipient,
                    amount: vec![Coin {
                        denom: native_denom,
                        amount: native_amount,
                    }],
                })
                .add_message(BankMsg::Send {
                    to_address: xerc20_recipient,
                    amount: vec![Coin {
                        denom: xerc20_denom,
                        amount: xerc20_amount,
                    }],
                })
                .add_attribute("action", "native_then_xerc20"))
        }
        ExecuteMsg::NativeSiblingThenHandledFailure {
            native_denom,
            native_recipient,
            native_amount,
            child_contract,
            child_recipient,
            child_amount,
            failing_native_amount,
            xerc20_denom,
        } => {
            deps.api.addr_validate(&native_recipient)?;
            deps.api.addr_validate(&child_contract)?;
            deps.api.addr_validate(&child_recipient)?;
            reset_reply_state(deps.storage);

            let child_msg = to_json_binary(&ExecuteMsg::ChildTransferThenFail {
                denom: xerc20_denom,
                recipient: child_recipient,
                amount: child_amount,
                native_denom: native_denom.clone(),
                native_amount: failing_native_amount,
            })?;

            Ok(Response::new()
                .add_message(BankMsg::Send {
                    to_address: native_recipient,
                    amount: vec![Coin {
                        denom: native_denom,
                        amount: native_amount,
                    }],
                })
                .add_submessage(SubMsg::reply_on_error(
                    WasmMsg::Execute {
                        contract_addr: child_contract,
                        msg: child_msg,
                        funds: vec![],
                    },
                    CHILD_FAILURE_REPLY_ID,
                ))
                .add_attribute("action", "native_sibling_then_handled_failure"))
        }
    }
}

#[entry_point]
pub fn reply(deps: DepsMut, _env: Env, reply: Reply) -> StdResult<Response> {
    match reply.id {
        OBSERVE_REPLY_ID => {
            if let SubMsgResult::Err(err) = reply.result {
                return Err(StdError::generic_err(format!(
                    "observed transfer failed: {err}"
                )));
            }

            let address = load_string(deps.as_ref(), OBSERVE_ADDRESS_KEY)?;
            let denom = load_string(deps.as_ref(), OBSERVE_DENOM_KEY)?;
            let balance = deps.querier.query_balance(address, denom)?;
            deps.storage
                .set(OBSERVED_AMOUNT_KEY, balance.amount.to_string().as_bytes());
            Ok(Response::new()
                .add_attribute("action", "observe_after_send")
                .add_attribute("observed_amount", balance.amount))
        }
        CHILD_FAILURE_REPLY_ID | HANDLED_SEND_REPLY_ID => {
            let error = match reply.result {
                SubMsgResult::Err(err) => err,
                SubMsgResult::Ok(_) => {
                    return Err(StdError::generic_err("expected submessage failure"));
                }
            };
            deps.storage.set(REPLY_CAUGHT_KEY, b"true");
            deps.storage.set(REPLY_ERROR_KEY, error.as_bytes());
            Ok(Response::new()
                .add_attribute("action", "handled_failure")
                .add_attribute("reply_id", reply.id.to_string()))
        }
        id => Err(StdError::generic_err(format!("unknown reply id {id}"))),
    }
}

#[entry_point]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Observation {} => {
            let amount = load_uint128(deps, OBSERVED_AMOUNT_KEY)?;
            to_json_binary(&ObservationResponse { amount })
        }
        QueryMsg::ReplyState {} => {
            let caught = deps
                .storage
                .get(REPLY_CAUGHT_KEY)
                .map(|value| value == b"true")
                .unwrap_or(false);
            let error = deps
                .storage
                .get(REPLY_ERROR_KEY)
                .map(String::from_utf8)
                .transpose()
                .map_err(|err| StdError::generic_err(err.to_string()))?
                .unwrap_or_default();
            to_json_binary(&ReplyStateResponse { caught, error })
        }
    }
}

fn reset_reply_state(storage: &mut dyn Storage) {
    storage.set(REPLY_CAUGHT_KEY, b"false");
    storage.remove(REPLY_ERROR_KEY);
}

fn load_string(deps: Deps, key: &[u8]) -> StdResult<String> {
    let value = deps
        .storage
        .get(key)
        .ok_or_else(|| StdError::not_found(String::from_utf8_lossy(key)))?;
    String::from_utf8(value).map_err(|err| StdError::generic_err(err.to_string()))
}

fn load_uint128(deps: Deps, key: &[u8]) -> StdResult<Uint128> {
    let value = load_string(deps, key)?;
    value
        .parse::<u128>()
        .map(Uint128::new)
        .map_err(|err| StdError::generic_err(err.to_string()))
}
