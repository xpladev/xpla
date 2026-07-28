use cosmwasm_std::{
    entry_point, to_json_binary, AnyMsg, Binary, CosmosMsg, Deps, DepsMut, Empty, Env, MessageInfo,
    Response, StdResult,
};
use serde::{Deserialize, Serialize};

#[derive(Serialize, Deserialize)]
pub struct InstantiateMsg {}

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum ExecuteMsg {
    DispatchAny { type_url: String, value: Binary },
}

#[derive(Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum QueryMsg {
    Version {},
}

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    _msg: InstantiateMsg,
) -> StdResult<Response> {
    Ok(Response::new().add_attribute("contract", "any-dispatch"))
}

#[entry_point]
pub fn execute(
    _deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    match msg {
        ExecuteMsg::DispatchAny { type_url, value } => Ok(Response::new()
            .add_message(CosmosMsg::<Empty>::Any(AnyMsg {
                type_url: type_url.clone(),
                value,
            }))
            .add_attribute("action", "dispatch_any")
            .add_attribute("type_url", type_url)),
    }
}

#[entry_point]
pub fn query(_deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Version {} => to_json_binary("any-dispatch-0.1.0"),
    }
}
