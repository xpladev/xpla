import hre from 'hardhat'
import { expect } from 'chai'
import {
  AUTH_PRECOMPILE_ADDRESS,
  BECH32_PRECOMPILE_ADDRESS,
  LARGE_GAS_LIMIT,
  findEvent,
} from '../common.js'

const { ethers } = await hre.network.connect()

const TYPE_URL_MSG_ETHEREUM_TX = '/cosmos.evm.vm.v1.MsgEthereumTx'
const TYPE_URL_MSG_EXEC = '/cosmos.authz.v1beta1.MsgExec'
const TYPE_URL_MSG_GRANT = '/cosmos.authz.v1beta1.MsgGrant'
const TYPE_URL_GENERIC_AUTHORIZATION =
  '/cosmos.authz.v1beta1.GenericAuthorization'
const TYPE_URL_GOV_MSG_SUBMIT_PROPOSAL =
  '/cosmos.gov.v1.MsgSubmitProposal'
const TYPE_URL_GROUP_MSG_SUBMIT_PROPOSAL =
  '/cosmos.group.v1.MsgSubmitProposal'
const BECH32_CHARSET = 'qpzry9x8gf2tvdw0s3jn54khce6mua7l'

const REST_URL = process.env.COSMOS_REST_URL || 'http://127.0.0.1:1317'
const JSON_RPC_URL = process.env.JSON_RPC_URL || 'http://127.0.0.1:8545'
const EXPECT_BASELINE = process.env.EXPECT_WASM_ANY_EVM_BASELINE === '1'
const RECEIPT_POLL_INTERVAL_MS = 500
const RECEIPT_POLL_ATTEMPTS = 80
const INNER_MARKER = ethers.keccak256(
  ethers.toUtf8Bytes('wasm any inner evm log')
)
const ERROR_IFACE = new ethers.Interface(['error Error(string)'])
const TEST_PRIVATE_KEYS = [
  '0x88CBEAD91AEE890D27BF06E003ADE3D4E952427E88F88D31D61D3EF5E5D54305',
  '0x3B7955D25189C99A7468192FCBC6429205C158834053EBE3F78F4512AB432DB9',
  '0xe9b1d63e8acd7fe676acb43afb390d4b0202dab61abec9cf2a561e4becb147de',
]

function bytesOf(value) {
  if (typeof value === 'string') {
    return ethers.getBytes(value)
  }
  return Uint8Array.from(value)
}

function bech32Polymod(values) {
  const generators = [
    0x3b6a57b2, 0x26508e6d, 0x1ea119fa, 0x3d4233dd, 0x2a1462b3,
  ]
  let checksum = 1

  for (const value of values) {
    const top = checksum >>> 25
    checksum = ((checksum & 0x1ffffff) << 5) ^ value
    for (let bit = 0; bit < generators.length; bit += 1) {
      if ((top >>> bit) & 1) {
        checksum ^= generators[bit]
      }
    }
  }

  return checksum >>> 0
}

function decodeBech32Address(address) {
  const normalized = address.toLowerCase()
  if (address !== normalized && address !== address.toUpperCase()) {
    throw new Error(`mixed-case Bech32 address: ${address}`)
  }

  const separator = normalized.lastIndexOf('1')
  if (separator <= 0 || separator + 7 > normalized.length) {
    throw new Error(`invalid Bech32 address: ${address}`)
  }

  const hrp = normalized.slice(0, separator)
  const values = [...normalized.slice(separator + 1)].map((char) => {
    const value = BECH32_CHARSET.indexOf(char)
    if (value < 0) {
      throw new Error(`invalid Bech32 character: ${char}`)
    }
    return value
  })
  const expandedHrp = [
    ...[...hrp].map((char) => char.charCodeAt(0) >>> 5),
    0,
    ...[...hrp].map((char) => char.charCodeAt(0) & 31),
  ]
  if (bech32Polymod([...expandedHrp, ...values]) !== 1) {
    throw new Error(`invalid Bech32 checksum: ${address}`)
  }

  const output = []
  let accumulator = 0
  let bits = 0
  for (const value of values.slice(0, -6)) {
    accumulator = ((accumulator << 5) | value) & 0xfff
    bits += 5
    while (bits >= 8) {
      bits -= 8
      output.push((accumulator >>> bits) & 0xff)
    }
  }
  if (bits >= 5 || ((accumulator << (8 - bits)) & 0xff) !== 0) {
    throw new Error(`invalid Bech32 padding: ${address}`)
  }

  return Uint8Array.from(output)
}

function concatBytes(...parts) {
  const arrays = parts.map(bytesOf)
  const out = new Uint8Array(arrays.reduce((sum, part) => sum + part.length, 0))
  let offset = 0
  for (const part of arrays) {
    out.set(part, offset)
    offset += part.length
  }
  return out
}

function encodeVarint(value) {
  let n = BigInt(value)
  const out = []
  while (n >= 0x80n) {
    out.push(Number((n & 0x7fn) | 0x80n))
    n >>= 7n
  }
  out.push(Number(n))
  return Uint8Array.from(out)
}

function fieldKey(fieldNumber, wireType) {
  return encodeVarint((BigInt(fieldNumber) << 3n) | BigInt(wireType))
}

function fieldBytes(fieldNumber, value) {
  const bytes = bytesOf(value)
  return concatBytes(fieldKey(fieldNumber, 2), encodeVarint(bytes.length), bytes)
}

function fieldString(fieldNumber, value) {
  return fieldBytes(fieldNumber, ethers.toUtf8Bytes(value))
}

function fieldVarint(fieldNumber, value) {
  return concatBytes(fieldKey(fieldNumber, 0), encodeVarint(value))
}

function encodeAny(typeUrl, value) {
  return concatBytes(fieldString(1, typeUrl), fieldBytes(2, value))
}

function encodeMsgEthereumTx(from, raw) {
  return concatBytes(fieldBytes(5, from), fieldBytes(6, raw))
}

function encodeMsgExec(grantee, msgs) {
  return concatBytes(
    fieldString(1, grantee),
    ...msgs.map((msg) => fieldBytes(2, msg))
  )
}

function encodeGenericAuthorization(msgTypeUrl) {
  return fieldString(1, msgTypeUrl)
}

function encodeMsgGrant(granter, grantee, authorizationTypeUrl) {
  const authorization = encodeAny(
    TYPE_URL_GENERIC_AUTHORIZATION,
    encodeGenericAuthorization(authorizationTypeUrl)
  )
  const grant = fieldBytes(1, authorization)
  return concatBytes(
    fieldString(1, granter),
    fieldString(2, grantee),
    fieldBytes(3, grant)
  )
}

function encodeGovMsgSubmitProposal({
  messages,
  proposer,
  title,
  summary,
}) {
  return concatBytes(
    ...messages.map((msg) => fieldBytes(1, msg)),
    fieldString(3, proposer),
    fieldString(5, title),
    fieldString(6, summary)
  )
}

function encodeGroupMsgSubmitProposal({
  groupPolicyAddress,
  proposers,
  metadata,
  messages,
  exec,
  title,
  summary,
}) {
  return concatBytes(
    fieldString(1, groupPolicyAddress),
    ...proposers.map((proposer) => fieldString(2, proposer)),
    fieldString(3, metadata),
    ...messages.map((msg) => fieldBytes(4, msg)),
    fieldVarint(5, exec),
    fieldString(6, title),
    fieldString(7, summary)
  )
}

function buildDispatchAnyMsg(typeUrl, value) {
  return ethers.toUtf8Bytes(
    JSON.stringify({
      dispatch_any: {
        type_url: typeUrl,
        value: Buffer.from(bytesOf(value)).toString('base64'),
      },
    })
  )
}

function moduleAddressBytes(name) {
  return ethers.getBytes(
    ethers.dataSlice(ethers.sha256(ethers.toUtf8Bytes(name)), 0, 20)
  )
}

function receiptLogsFrom(receipt, address) {
  const normalized = address.toLowerCase()
  return receipt.logs.filter((log) => log.address.toLowerCase() === normalized)
}

async function providerLogsFrom(receipt, address) {
  return ethers.provider.getLogs({
    address,
    fromBlock: receipt.blockNumber,
    toBlock: receipt.blockNumber,
  })
}

async function resolveWasmAddress(addr) {
  const converter = await ethers.getContractAt(
    'Bech32I',
    BECH32_PRECOMPILE_ADDRESS
  )
  const auth = await ethers.getContractAt('IAuth', AUTH_PRECOMPILE_ADDRESS)

  const hex = addr.startsWith('xpla')
    ? await converter.bech32ToHex.staticCall(addr)
    : addr
  const bech32 = await auth.account.staticCall(hex)
  if (!bech32) {
    throw new Error(`no Cosmos account found for Wasm EVM address ${hex}`)
  }

  const bytes = decodeBech32Address(bech32)
  if (bytes.length !== 32) {
    throw new Error(
      `expected a 32-byte canonical Wasm account for ${hex}, got ${bytes.length}`
    )
  }
  const mappedHex = ethers.hexlify(bytes.slice(-20))
  if (mappedHex.toLowerCase() !== hex.toLowerCase()) {
    throw new Error(
      `canonical Wasm account ${bech32} does not map back to ${hex}`
    )
  }

  return { bech32, bytes, hex }
}

async function toBech32(address) {
  const converter = await ethers.getContractAt(
    'Bech32I',
    BECH32_PRECOMPILE_ADDRESS
  )
  return converter.hexToBech32.staticCall(address, 'xpla')
}

async function signEmitterTx(innerSigner, emitter, value) {
  const feeData = await ethers.provider.getFeeData()
  const network = await ethers.provider.getNetwork()
  const raw = await innerSigner.signTransaction({
    chainId: network.chainId,
    type: 0,
    nonce: await ethers.provider.getTransactionCount(innerSigner.address),
    gasLimit: 500_000n,
    gasPrice: feeData.gasPrice ?? 1_000_000_000n,
    to: await emitter.getAddress(),
    data: emitter.interface.encodeFunctionData('emitInner', [
      INNER_MARKER,
      value,
    ]),
    value: 0n,
  })

  return {
    raw,
    hash: ethers.keccak256(raw),
  }
}

async function queryJson(path, params = {}) {
  const url = new URL(path, REST_URL)
  for (const [key, value] of Object.entries(params)) {
    url.searchParams.set(key, value)
  }

  const response = await fetch(url)
  if (!response.ok) {
    throw new Error(
      `REST query failed: ${response.status} ${response.statusText} ${url}`
    )
  }
  return response.json()
}

async function queryCosmosTxsByEthereumHash(hash) {
  const data = await queryJson('/cosmos/tx/v1beta1/txs', {
    query: `ethereum_tx.ethereumTxHash='${hash}'`,
  })
  return data?.tx_responses ?? []
}

async function waitForOuterReceiptHash(hash, label) {
  for (let attempt = 0; attempt < RECEIPT_POLL_ATTEMPTS; attempt += 1) {
    const receipt = await ethers.provider.getTransactionReceipt(hash)
    if (receipt) {
      return receipt
    }

    const cosmosResponses = await queryCosmosTxsByEthereumHash(hash)
    const failed = cosmosResponses.find(
      (response) => Number(response.code ?? 0) !== 0
    )
    if (failed) {
      const rawLog = String(failed.raw_log ?? 'unknown Cosmos error')
        .split('\n', 1)[0]
      throw new Error(
        `${label} failed in Cosmos tx ${failed.txhash} ` +
          `(code ${failed.code}): ${rawLog}`
      )
    }

    await new Promise((resolve) =>
      setTimeout(resolve, RECEIPT_POLL_INTERVAL_MS)
    )
  }

  throw new Error(`${label} receipt timed out for ${hash}`)
}

async function waitForOuterReceipt(tx, label) {
  return waitForOuterReceiptHash(tx.hash, label)
}

async function sendBaselineAnyCall({
  caller,
  wrapper,
  method,
  wasmMsg,
  wasmAddress,
  label,
}) {
  const feeData = await ethers.provider.getFeeData()
  const network = await ethers.provider.getNetwork()
  const request = await wrapper[method].populateTransaction(
    wasmAddress,
    wasmMsg
  )
  const raw = await caller.signTransaction({
    ...request,
    chainId: network.chainId,
    type: 0,
    nonce: await ethers.provider.getTransactionCount(caller.address),
    gasLimit: LARGE_GAS_LIMIT,
    gasPrice: feeData.gasPrice ?? 1_000_000_000n,
  })
  const hash = ethers.keccak256(raw)
  const controller = new AbortController()
  const broadcastFailure = fetch(JSON_RPC_URL, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({
      jsonrpc: '2.0',
      id: 1,
      method: 'eth_sendRawTransaction',
      params: [raw],
    }),
    signal: controller.signal,
  })
    .then(async (response) => {
      const result = await response.json()
      if (!response.ok || result.error) {
        throw new Error(
          `${label} broadcast failed: ${
            result.error?.message ?? response.statusText
          }`
        )
      }
      return new Promise(() => {})
    })
    .catch((error) => {
      if (controller.signal.aborted) {
        return new Promise(() => {})
      }
      throw error
    })

  try {
    const receipt = await Promise.race([
      waitForOuterReceiptHash(hash, label),
      broadcastFailure,
    ])
    return { hash, receipt }
  } finally {
    controller.abort()
  }
}

async function queryAuthzGrants(granter, grantee) {
  const url = new URL('/cosmos/authz/v1beta1/grants', REST_URL)
  url.searchParams.set('granter', granter)
  url.searchParams.set('grantee', grantee)
  url.searchParams.set('msg_type_url', TYPE_URL_MSG_ETHEREUM_TX)

  const response = await fetch(url)
  const data = await response.json()
  if (!response.ok) {
    if (
      data?.code === 2 &&
      String(data?.message ?? '').includes('authorization not found')
    ) {
      return []
    }
    throw new Error(
      `REST query failed: ${response.status} ${response.statusText} ${url}`
    )
  }
  return data?.grants ?? []
}

async function queryGovProposalIds() {
  const data = await queryJson('/cosmos/gov/v1/proposals')
  return new Set((data?.proposals ?? []).map((proposal) => String(proposal.id)))
}

function txEvents(tx) {
  return tx.events ?? tx.tx_response?.events ?? []
}

function hasCosmosEventAttribute(txResponses, eventType, key, value) {
  const normalizedValue = String(value).toLowerCase()
  return txResponses.some((tx) =>
    txEvents(tx).some(
      (event) =>
        event.type === eventType &&
        (event.attributes ?? []).some(
          (attr) =>
            attr.key === key &&
            String(attr.value ?? '').toLowerCase() === normalizedValue
        )
    )
  )
}

function getDispatchResult(receipt, wrapper) {
  const result = findEvent(receipt.logs, wrapper.interface, 'AnyDispatchResult')
  expect(result).to.exist
  return result
}

function describeReturnData(data, wrapper) {
  if (!data || data === '0x') {
    return 'empty returnData'
  }

  try {
    const parsed = wrapper.interface.parseError(data)
    return `${parsed.name}(${parsed.args.map((arg) => String(arg)).join(',')})`
  } catch {}

  try {
    const parsed = ERROR_IFACE.parseError(data)
    return `${parsed.name}(${parsed.args.map((arg) => String(arg)).join(',')})`
  } catch {}

  try {
    const text = ethers.toUtf8String(data)
    if (text) {
      return `${data} ${text}`
    }
  } catch {}

  return data
}

function expectDisabledMsgGuard(
  result,
  label,
  wrapper,
  expectedTypeUrl = TYPE_URL_MSG_ETHEREUM_TX
) {
  expect(
    result.args.success,
    `${label}: ${describeReturnData(result.args.returnData, wrapper)}`
  ).to.equal(false)

  const details = describeReturnData(result.args.returnData, wrapper)
  expect(details, `${label} must fail on the disabled MsgEthereumTx guard`).to
    .include('found disabled msg type')
  expect(details, `${label} must name the disabled msg type`).to.include(
    expectedTypeUrl
  )
}

function expectSignerOrGuardRejection(result, label, wrapper) {
  expect(
    result.args.success,
    `${label}: ${describeReturnData(result.args.returnData, wrapper)}`
  ).to.equal(false)

  const details = describeReturnData(result.args.returnData, wrapper)
  expect(
    details.includes('found disabled msg type') ||
      details.includes("contract doesn't have permission"),
    `${label} must fail before inner EVM execution: ${details}`
  ).to.equal(true)
}

async function blockContainsEthereumTxHash(blockNumber, hash) {
  const block = await ethers.provider.send('eth_getBlockByNumber', [
    ethers.toQuantity(blockNumber),
    true,
  ])
  const normalized = hash.toLowerCase()
  return (block?.transactions ?? []).some((tx) => {
    const txHash = typeof tx === 'string' ? tx : tx.hash
    return String(txHash ?? '').toLowerCase() === normalized
  })
}

async function assertNoInnerEvmSurface({
  receipt,
  emitter,
  innerHash,
  outerHash,
}) {
  const emitterAddress = await emitter.getAddress()
  expect(receiptLogsFrom(receipt, emitterAddress)).to.have.length(0)
  expect(await providerLogsFrom(receipt, emitterAddress)).to.have.length(0)
  expect(
    await ethers.provider.send('eth_getTransactionReceipt', [innerHash])
  ).to.equal(null)
  expect(await ethers.provider.getTransaction(innerHash)).to.equal(null)
  expect(await blockContainsEthereumTxHash(receipt.blockNumber, innerHash)).to
    .equal(false)
  expect(await queryCosmosTxsByEthereumHash(innerHash)).to.have.length(0)

  if (outerHash) {
    const outerResponses = await queryCosmosTxsByEthereumHash(outerHash)
    expect(
      hasCosmosEventAttribute(
        outerResponses,
        'ethereum_tx',
        'ethereumTxHash',
        innerHash
      ),
      'outer Cosmos tx events must not contain the reverted inner EVM hash'
    ).to.equal(false)
  }
}

async function assertInnerEvmExecuted({ receipt, emitter, innerHash, value }) {
  const emitterAddress = await emitter.getAddress()
  expect(await emitter.counter()).to.equal(value)
  expect(receiptLogsFrom(receipt, emitterAddress)).not.to.have.length(0)
  expect(await providerLogsFrom(receipt, emitterAddress)).not.to.have.length(0)

  const indexedResponses = await queryCosmosTxsByEthereumHash(innerHash)
  const indexedReceipt =
    (await ethers.provider.send('eth_getTransactionReceipt', [innerHash])) !==
    null
  const indexedTransaction =
    (await ethers.provider.getTransaction(innerHash)) !== null
  const indexedBlock = await blockContainsEthereumTxHash(
    receipt.blockNumber,
    innerHash
  )
  expect(
    indexedResponses.length > 0 ||
      indexedReceipt ||
      indexedTransaction ||
      indexedBlock,
    'inner Ethereum tx must be visible on at least one transaction index surface'
  ).to.equal(true)
}

async function deployFixture() {
  const outerCaller = new ethers.Wallet(TEST_PRIVATE_KEYS[0], ethers.provider)
  const innerSigner = new ethers.Wallet(TEST_PRIVATE_KEYS[1], ethers.provider)
  const authzGrantee = new ethers.Wallet(TEST_PRIVATE_KEYS[2], ethers.provider)

  const Emitter = await ethers.getContractFactory('InnerEvmEmitter')
  const emitter = await Emitter.deploy()
  await emitter.waitForDeployment()

  const Wrapper = await ethers.getContractFactory('WasmAnyEvmPoC')
  const wrapper = await Wrapper.deploy()
  await wrapper.waitForDeployment()

  return {
    outerCaller,
    innerSigner,
    authzGrantee,
    emitter,
    wrapper,
  }
}

describe('wasm CosmosMsg::Any MsgEthereumTx guard', function () {
  let anyDispatchWasm

  before(async function () {
    const anyDispatchAddr = process.env.ANY_DISPATCH_WASM_ADDRESS
    if (!anyDispatchAddr) {
      throw new Error('ANY_DISPATCH_WASM_ADDRESS is required')
    }

    anyDispatchWasm = await resolveWasmAddress(anyDispatchAddr)
  })

  async function buildDirectEthereumWasmMsg(
    innerSigner,
    emitter,
    value,
    from = anyDispatchWasm.bytes
  ) {
    const signed = await signEmitterTx(innerSigner, emitter, value)
    const msgEthereumTx = encodeMsgEthereumTx(
      ethers.getBytes(from),
      ethers.getBytes(signed.raw)
    )
    return {
      innerHash: signed.hash,
      wasmMsg: buildDispatchAnyMsg(TYPE_URL_MSG_ETHEREUM_TX, msgEthereumTx),
    }
  }

  async function buildAuthzEthereumWasmMsg(innerSigner, emitter, value) {
    const signed = await signEmitterTx(innerSigner, emitter, value)
    const msgEthereumTx = encodeAny(
      TYPE_URL_MSG_ETHEREUM_TX,
      encodeMsgEthereumTx(
        anyDispatchWasm.bytes,
        ethers.getBytes(signed.raw)
      )
    )
    const msgExec = encodeMsgExec(anyDispatchWasm.bech32, [msgEthereumTx])

    return {
      innerHash: signed.hash,
      wasmMsg: buildDispatchAnyMsg(TYPE_URL_MSG_EXEC, msgExec),
    }
  }

  async function grantMsgEthereumTxAuthz(wrapper, grantee) {
    const msgGrant = encodeMsgGrant(
      anyDispatchWasm.bech32,
      grantee,
      TYPE_URL_MSG_ETHEREUM_TX
    )
    const wasmMsg = buildDispatchAnyMsg(TYPE_URL_MSG_GRANT, msgGrant)
    const tx = await wrapper.tryExecuteAny(anyDispatchWasm.hex, wasmMsg, {
      gasLimit: LARGE_GAS_LIMIT,
    })
    const receipt = await waitForOuterReceipt(tx, 'MsgGrant dispatch')
    return getDispatchResult(receipt, wrapper)
  }

  async function expectBlockedOrBaselineExecution({
    label,
    outerCaller,
    wrapper,
    emitter,
    wasmMsg,
    innerHash,
    value,
  }) {
    let tx
    let receipt
    if (EXPECT_BASELINE) {
      const baseline = await sendBaselineAnyCall({
        caller: outerCaller,
        wrapper,
        method: 'tryExecuteAny',
        wasmMsg,
        wasmAddress: anyDispatchWasm.hex,
        label,
      })
      tx = { hash: baseline.hash }
      receipt = baseline.receipt
    } else {
      tx = await wrapper.tryExecuteAny(anyDispatchWasm.hex, wasmMsg, {
        gasLimit: LARGE_GAS_LIMIT,
      })
      receipt = await waitForOuterReceipt(tx, label)
    }
    const result = getDispatchResult(receipt, wrapper)

    if (EXPECT_BASELINE) {
      expect(
        result.args.success,
        `${label} baseline dispatch: ${describeReturnData(
          result.args.returnData,
          wrapper
        )}`
      ).to.equal(true)
      await assertInnerEvmExecuted({ receipt, emitter, innerHash, value })
      return
    }

    expectDisabledMsgGuard(result, label, wrapper)
    expect(await emitter.counter()).to.equal(0n)
    await assertNoInnerEvmSurface({
      receipt,
      emitter,
      innerHash,
      outerHash: tx.hash,
    })
  }

  it('R2 rejects direct wasm Any MsgEthereumTx before inner EVM execution', async function () {
    const { outerCaller, innerSigner, emitter, wrapper } = await deployFixture()
    const value = 11n
    const { wasmMsg, innerHash } = await buildDirectEthereumWasmMsg(
      innerSigner,
      emitter,
      value
    )

    await expectBlockedOrBaselineExecution({
      label: 'direct MsgEthereumTx',
      outerCaller,
      wrapper,
      emitter,
      wasmMsg,
      innerHash,
      value,
    })
  })

  it('R2 negative control rejects direct wasm Any MsgEthereumTx when From is not the wasm contract', async function () {
    const { innerSigner, emitter, wrapper } = await deployFixture()
    const value = 12n
    const { wasmMsg, innerHash } = await buildDirectEthereumWasmMsg(
      innerSigner,
      emitter,
      value,
      innerSigner.address
    )

    const tx = await wrapper.tryExecuteAny(anyDispatchWasm.hex, wasmMsg, {
      gasLimit: LARGE_GAS_LIMIT,
    })
    const receipt = await waitForOuterReceipt(
      tx,
      'direct MsgEthereumTx negative control'
    )
    const result = getDispatchResult(receipt, wrapper)

    expectSignerOrGuardRejection(
      result,
      'direct MsgEthereumTx with non-contract From',
      wrapper
    )
    expect(await emitter.counter()).to.equal(0n)
    await assertNoInnerEvmSurface({
      receipt,
      emitter,
      innerHash,
      outerHash: tx.hash,
    })
  })

  it('R3 rejects wasm Any authz MsgExec containing MsgEthereumTx', async function () {
    const { outerCaller, innerSigner, emitter, wrapper } = await deployFixture()
    const value = 13n

    const { wasmMsg, innerHash } = await buildAuthzEthereumWasmMsg(
      innerSigner,
      emitter,
      value
    )

    await expectBlockedOrBaselineExecution({
      label: 'authz MsgExec MsgEthereumTx',
      outerCaller,
      wrapper,
      emitter,
      wasmMsg,
      innerHash,
      value,
    })
  })

  it('R4 rejects wasm Any authz MsgGrant for MsgEthereumTx before persistence', async function () {
    const { authzGrantee, wrapper } = await deployFixture()
    const grantee = await toBech32(authzGrantee.address)
    const result = await grantMsgEthereumTxAuthz(wrapper, grantee)
    const grants = await queryAuthzGrants(anyDispatchWasm.bech32, grantee)

    if (EXPECT_BASELINE) {
      expect(result.args.success, 'MsgGrant baseline dispatch').to.equal(true)
      expect(grants).not.to.have.length(0)
      return
    }

    expectDisabledMsgGuard(result, 'authz MsgGrant MsgEthereumTx', wrapper)
    expect(grants).to.have.length(0)
  })

  it('R5 rejects wasm Any gov proposal carrying MsgEthereumTx before persistence', async function () {
    const { innerSigner, emitter, wrapper } = await deployFixture()
    const proposalIdsBefore = await queryGovProposalIds()

    const signed = await signEmitterTx(innerSigner, emitter, 17n)
    const innerMsg = encodeAny(
      TYPE_URL_MSG_ETHEREUM_TX,
      encodeMsgEthereumTx(moduleAddressBytes('gov'), ethers.getBytes(signed.raw))
    )
    const proposal = encodeGovMsgSubmitProposal({
      messages: [innerMsg],
      proposer: anyDispatchWasm.bech32,
      title: 'blocked evm proposal',
      summary:
        'proposal containing MsgEthereumTx must not be accepted from wasm Any',
    })
    const wasmMsg = buildDispatchAnyMsg(
      TYPE_URL_GOV_MSG_SUBMIT_PROPOSAL,
      proposal
    )

    const tx = await wrapper.tryExecuteAny(anyDispatchWasm.hex, wasmMsg, {
      gasLimit: LARGE_GAS_LIMIT,
    })
    const receipt = await waitForOuterReceipt(tx, 'gov proposal dispatch')
    const result = getDispatchResult(receipt, wrapper)
    const proposalIdsAfter = await queryGovProposalIds()
    const submitted = [...proposalIdsAfter].some(
      (proposalId) => !proposalIdsBefore.has(proposalId)
    )

    if (EXPECT_BASELINE) {
      expect(
        result.args.success,
        `gov proposal baseline dispatch: ${describeReturnData(
          result.args.returnData,
          wrapper
        )}`
      ).to.equal(true)
      expect(submitted, 'baseline must persist the proposal').to.equal(true)
      return
    }

    expectDisabledMsgGuard(result, 'gov proposal MsgEthereumTx', wrapper)
    expect(submitted, 'rejected proposal must not be persisted').to.equal(false)
  })

  it('R5 rejects an unregistered wasm Any group proposal before routing', async function () {
    if (EXPECT_BASELINE) {
      this.skip()
      return
    }

    const { innerSigner, emitter, wrapper } = await deployFixture()
    const signed = await signEmitterTx(innerSigner, emitter, 18n)
    const innerMsg = encodeAny(
      TYPE_URL_MSG_ETHEREUM_TX,
      encodeMsgEthereumTx(
        moduleAddressBytes('group'),
        ethers.getBytes(signed.raw)
      )
    )
    const proposal = encodeGroupMsgSubmitProposal({
      groupPolicyAddress: anyDispatchWasm.bech32,
      proposers: [anyDispatchWasm.bech32],
      metadata: 'blocked evm group proposal',
      messages: [innerMsg],
      exec: 1,
      title: 'blocked evm group proposal',
      summary:
        'an unregistered group proposal must fail closed during Any decoding',
    })
    const wasmMsg = buildDispatchAnyMsg(
      TYPE_URL_GROUP_MSG_SUBMIT_PROPOSAL,
      proposal
    )

    const tx = await wrapper.tryExecuteAny(anyDispatchWasm.hex, wasmMsg, {
      gasLimit: LARGE_GAS_LIMIT,
    })
    const receipt = await waitForOuterReceipt(tx, 'group proposal dispatch')
    const result = getDispatchResult(receipt, wrapper)

    expect(
      result.args.success,
      `unregistered group proposal: ${describeReturnData(
        result.args.returnData,
        wrapper
      )}`
    ).to.equal(false)
    expect(await emitter.counter()).to.equal(0n)
    await assertNoInnerEvmSurface({
      receipt,
      emitter,
      innerHash: signed.hash,
      outerHash: tx.hash,
    })
  })

  it('R6 removes inner EVM state, logs, and index surfaces when the outer EVM frame reverts', async function () {
    const { outerCaller, innerSigner, emitter, wrapper } = await deployFixture()
    const value = 19n

    const { wasmMsg, innerHash } = await buildAuthzEthereumWasmMsg(
      innerSigner,
      emitter,
      value
    )

    let tx
    let receipt
    if (EXPECT_BASELINE) {
      const baseline = await sendBaselineAnyCall({
        caller: outerCaller,
        wrapper,
        method: 'tryExecuteAnyThenRevert',
        wasmMsg,
        wasmAddress: anyDispatchWasm.hex,
        label: 'outer-reverted authz dispatch',
      })
      tx = { hash: baseline.hash }
      receipt = baseline.receipt
    } else {
      tx = await wrapper.tryExecuteAnyThenRevert(
        anyDispatchWasm.hex,
        wasmMsg,
        { gasLimit: LARGE_GAS_LIMIT }
      )
      receipt = await waitForOuterReceipt(
        tx,
        'outer-reverted authz dispatch'
      )
    }
    const result = getDispatchResult(receipt, wrapper)
    expect(result.args.success).to.equal(false)

    if (EXPECT_BASELINE) {
      const forcedRevert = wrapper.interface.parseError(result.args.returnData)
      expect(forcedRevert.name).to.equal('ForcedOuterRevertAfterAny')

      const emitterAddress = await emitter.getAddress()
      const leakedReceiptLogs = receiptLogsFrom(receipt, emitterAddress).length
      const leakedProviderLogs = (await providerLogsFrom(receipt, emitterAddress))
        .length
      const indexedResponses = await queryCosmosTxsByEthereumHash(innerHash)
      const indexedTransaction =
        (await ethers.provider.getTransaction(innerHash)) !== null
      const indexedBlock = await blockContainsEthereumTxHash(
        receipt.blockNumber,
        innerHash
      )
      const outerResponses = await queryCosmosTxsByEthereumHash(tx.hash)
      const leakedOuterEvent = hasCosmosEventAttribute(
        outerResponses,
        'ethereum_tx',
        'ethereumTxHash',
        innerHash
      )

      expect(
        (await emitter.counter()) === value ||
          leakedReceiptLogs > 0 ||
          leakedProviderLogs > 0 ||
          indexedResponses.length > 0 ||
          indexedTransaction ||
          indexedBlock ||
          leakedOuterEvent,
        'baseline must expose the reverted inner EVM execution on at least one surface'
      ).to.equal(true)
      return
    }

    expectDisabledMsgGuard(
      result,
      'outer-reverted authz MsgExec MsgEthereumTx',
      wrapper
    )
    expect(await emitter.counter()).to.equal(0n)
    await assertNoInnerEvmSurface({
      receipt,
      emitter,
      innerHash,
      outerHash: tx.hash,
    })
  })
})
