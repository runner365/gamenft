<script setup lang="ts">
import { ref, reactive, onMounted, onUnmounted, watch } from 'vue'
import TetrisGame from './components/TetrisGame.vue'
import ScoreBoard from './components/ScoreBoard.vue'
import { apiGet, apiPost, clearToken, getToken, isLoggedIn, onUnauthorized, setToken } from './api'
import { log } from './logger'

// Auto-logout on 401
onUnauthorized(() => {
  wsDisconnect()
  clearToken()
  loggedIn.value = false
  loggedInUser.value = ''
  addressError.value = 'Session expired, please log in again'
})

const contracts = reactive({
  game_token: '',
  game_items: '',
  marketplace: '',
  uniswap_v2_factory: '',
  uniswap_v2_router: '',
  weth: '',
  uniswap_liquidity_setup: '',
})

async function fetchContracts() {
  try {
    const res = await apiGet('/contracts')
    if (res.ok) {
      const data = await res.json()
      Object.assign(contracts, data)
      log.info('Contract addresses loaded from server')
      return
    }
  } catch (e) {
    log.warn('fetch contracts failed, will retry:', e)
  }
  setTimeout(fetchContracts, 5000)
}

// WebSocket connection for real-time notifications
let ws: WebSocket | null = null
let wsReconnectTimer: ReturnType<typeof setTimeout> | null = null

function wsConnect() {
  const token = getToken()
  if (!token) return

  const proto = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const url = `${proto}://${window.location.host}/ws?token=${encodeURIComponent(token)}`

  try {
    ws = new WebSocket(url)
  } catch (e) {
    log.warn('WS connect failed:', e)
    scheduleWsReconnect()
    return
  }

  ws.onopen = () => {
    log.info('WS connected')
  }

  ws.onmessage = (event) => {
    try {
      const msg = JSON.parse(event.data)
      log.info('WS message:', msg.type, msg)
      if (msg.type === 'reward_confirmed') {
        showToast(`${msg.item_type} mint confirmed on-chain`, 'success')
        addToLog(`Reward minted on-chain: ${msg.item_type}${msg.tx_hash ? ', tx=' + msg.tx_hash.slice(0, 10) + '...' : ''}`)
        loadInventory()
      } else if (msg.type === 'reward_failed') {
        showToast(`${msg.item_type} mint failed: ${msg.error}`, 'error')
        loadInventory()
      } else if (msg.type === 'item_listed') {
        showToast(`${msg.item_type} listed on marketplace`, 'success')
        loadInventory()
      } else if (msg.type === 'item_bought') {
        showToast(`You bought ${msg.quantity}x ${msg.item_type}`, 'success')
        loadInventory()
        loadMarketplace(marketPage.value)
      } else if (msg.type === 'item_sold') {
        showToast(`Your ${msg.quantity}x ${msg.item_type} was sold`, 'success')
        loadInventory()
        loadMarketplace(marketPage.value)
      }
    } catch (e) {
      log.warn('WS: parse error:', e)
    }
  }

  ws.onclose = () => {
    log.info('WS disconnected, will reconnect')
    ws = null
    scheduleWsReconnect()
  }

  ws.onerror = () => {
    log.warn('WS connection error')
    ws?.close()
  }
}

function wsDisconnect() {
  if (wsReconnectTimer) {
    clearTimeout(wsReconnectTimer)
    wsReconnectTimer = null
  }
  if (ws) {
    ws.onclose = null
    ws.close()
    ws = null
  }
}

function scheduleWsReconnect() {
  if (wsReconnectTimer) return
  wsReconnectTimer = setTimeout(() => {
    wsReconnectTimer = null
    if (isLoggedIn()) wsConnect()
  }, 5000)
}

// Toast notification
const toastMessage = ref('')
const toastType = ref<'success' | 'error'>('success')
let toastTimer: ReturnType<typeof setTimeout> | null = null

function showToast(msg: string, type: 'success' | 'error' = 'success') {
  toastMessage.value = msg
  toastType.value = type
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toastMessage.value = '' }, 5000)
}

const score = ref(0)
const knife = ref(0)
const pistol = ref(0)
const bomb = ref(0)
const tetrisRef = ref<any>(null)
const ethAddress = ref('')
const loggedIn = ref(false)
const loggedInUser = ref<string>('')
const addressError = ref('')
// sell modal
const sellModal = ref(false)
const sellItem = ref('')
const sellAmount = ref(1)
const sellPrice = ref('1000000000000000000') // default 1 tokenG
const sellMax = ref(0)
// buy tokens modal
const buyModal = ref(false)
const buyEthAmount = ref('0.01')
const buyEstimatedGmtk = ref('')
const buyLoading = ref(false)
const buyError = ref('')
const gameTokenAddress = ref('')
const poolReserveWeth = ref('0')
const poolReserveGmtk = ref('0')
// profile query
const queryAddress = ref('')
const profileModal = ref(false)
const profileUser = ref<any>(null)
const profileOwned = ref<any[]>([])
const profileListed = ref<any[]>([])
const profilePlayerItems = ref<any[]>([])
const profileTokenBalance = ref('')
const profileError = ref('')
// marketplace view
const viewMode = ref<'game' | 'market'>('game')
const activityLog = ref<string[]>([])
const MAX_LOG = 200

function addToLog(message: string) {
  const ts = new Date().toISOString().slice(11, 23)
  activityLog.value.push(`[${ts}] ${message}`)
  if (activityLog.value.length > MAX_LOG) {
    activityLog.value = activityLog.value.slice(-MAX_LOG)
  }
}

const marketItems = ref<any[]>([])
const marketTotal = ref(0)
const marketPage = ref(1)
const marketLoading = ref(false)
const marketError = ref('')
const buyItemId = ref<number | null>(null)
const buyItemLoading = ref(false)
const buyItemError = ref('')
function parsePrice(price: string): bigint {
  return BigInt(price.split('.')[0] || '0')
}

const PAGE_SIZE = 20

onMounted(() => {
  fetchContracts()
  if (isLoggedIn()) {
    loggedIn.value = true
    loadInventory()
    wsConnect()
  }
})

onUnmounted(() => {
  wsDisconnect()
})

function onScoreUpdate(s: number) { score.value = s }
function onReward(item: string) {
  if (item === 'knife') knife.value++
  else if (item === 'pistol') pistol.value++
  else if (item === 'bomb') bomb.value++

  if (item !== 'none') {
    showToast(`Earned ${item}! Minting on-chain...`, 'success')
  }

  if (isLoggedIn() && item !== 'none') {
    apiPost('/rewards', { item_type: item }).catch(err => log.warn('persist reward failed:', err))
  }
}

function startGame() {
  tetrisRef.value?.start?.()
}

function pauseGame() {
  tetrisRef.value?.pause?.()
}

function onUse(item: string) {
  if (item === 'knife' && knife.value > 0) { knife.value--; tetrisRef.value?.useItem?.(item) }
  else if (item === 'pistol' && pistol.value > 0) { pistol.value--; tetrisRef.value?.useItem?.(item) }
  else if (item === 'bomb' && bomb.value > 0) { bomb.value--; tetrisRef.value?.useItem?.(item) }
  else return
  apiPost('/rewards/use', { item_type: item }).catch(err => log.warn('use item persist failed:', err))
}

function onSell(item: string) {
  if (!isLoggedIn()) return
  tetrisRef.value?.pause?.()
  sellItem.value = item
  sellAmount.value = 1
  sellPrice.value = '1000000000000000000'
  if (item === 'knife') sellMax.value = knife.value
  else if (item === 'pistol') sellMax.value = pistol.value
  else if (item === 'bomb') sellMax.value = bomb.value
  sellModal.value = true
}

async function confirmSell() {
  const itemType = sellItem.value
  const amount = sellAmount.value
  const price = sellPrice.value
  if (amount <= 0 || amount > sellMax.value) return
  sellModal.value = false
  try {
    // 1. Backend validation only (DB update handled by ItemListed event listener)
    const res = await apiPost('/rewards/sell', { item_type: itemType, amount, price })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      log.warn('sell validation failed:', body.error || res.statusText)
      return
    }
    log.info('Sell validation passed')

    // 2. On-chain listing via Phantom
    const tokenID = itemType === 'knife' ? 1 : itemType === 'pistol' ? 2 : 3
    try {
      const eth = getProvider()
      if (eth) {
        const accounts: string[] = await eth.request({ method: 'eth_requestAccounts' })
        const from = accounts[0]

        // step 1: setApprovalForAll
        const approvalData = '0xa22cb465' +
          contracts.marketplace.toLowerCase().replace('0x', '').padStart(64, '0') +
          '0000000000000000000000000000000000000000000000000000000000000001'
        log.info('Listing on-chain: setApprovalForAll')
        const approveTx: string = await eth.request({
          method: 'eth_sendTransaction',
          params: [{ from, to: contracts.game_items, data: approvalData }]
        })
        await waitForTxReceipt(eth, approveTx)
        log.info('setApprovalForAll confirmed:', approveTx)

        // step 2: listERC1155
        const listData = '0xe8d9b52f' +
          contracts.game_items.toLowerCase().replace('0x', '').padStart(64, '0') +
          BigInt(tokenID).toString(16).padStart(64, '0') +
          BigInt(amount).toString(16).padStart(64, '0') +
          BigInt(price).toString(16).padStart(64, '0')
        log.info('Listing on-chain: listERC1155 nftContract:', contracts.game_items, 'tokenId:', tokenID, 'amount:', amount, 'price:', price)
        const listTx: string = await eth.request({
          method: 'eth_sendTransaction',
          params: [{ from, to: contracts.marketplace, data: listData }]
        })
        await waitForTxReceipt(eth, listTx)
        log.info('listERC1155 confirmed:', listTx)

        addToLog(`List item on-chain: ${itemType} x${amount} at price ${price}, tx=${listTx.slice(0, 10)}...`)
        // 3. Optimistic UI update + delayed sync with event listener
        if (itemType === 'knife') knife.value -= amount
        else if (itemType === 'pistol') pistol.value -= amount
        else if (itemType === 'bomb') bomb.value -= amount
        setTimeout(() => loadInventory(), 3000)
      }
    } catch (chainErr: any) {
      log.warn('on-chain listing failed:', chainErr?.message || chainErr)
    }
  } catch (err) {
    log.warn('sell error:', err)
  }
}

function cancelSell() {
  sellModal.value = false
}

async function login() {
  const v = ethAddress.value && ethAddress.value.trim()
  const re = /^0x[a-fA-F0-9]{40}$/
  if (!v || !re.test(v)) {
    addressError.value = '请输入有效的以 0x 开头的 40 字节十六进制地址'
    return
  }
  addressError.value = ''

  try {
    // 1) request challenge message (relative path, Vite proxy → backend)
    log.info('Requesting login challenge for address', v)
    const chRes = await apiGet(`/auth/challenge`, { address: v })
    if (!chRes.ok) {
      const err = await chRes.json().catch(() => ({ error: chRes.statusText }))
      addressError.value = err.error || '获取 challenge 失败'
      return
    }
    const chBody = await chRes.json()

    log.info('Challenge message:', JSON.stringify(chBody))

    const message = chBody.message

    // 2) check wallet
    const eth: any = getProvider()
    if (!eth) {
      addressError.value = '未检测到以太坊钱包（Phantom / MetaMask / OKX 等）'
      return
    }

    // request account access
    await eth.request?.({ method: 'eth_requestAccounts' })

    // 3) sign message with wallet
    // Try standard EIP-191 params order first: [message, address]
    let signature: string
    try {
      signature = await eth.request({ method: 'personal_sign', params: [message, v] })
    } catch (e: any) {
      // Only retry with reversed params if the error is a user rejection
      // (user cancelled), don't retry silently
      if (e?.code === 4001 || e?.code === 'ACTION_REJECTED') {
        addressError.value = '用户取消了签名'
        return
      }
      // Try reversed params (older wallet convention)
      try {
        signature = await eth.request({ method: 'personal_sign', params: [v, message] })
      } catch (e2: any) {
        if (e2?.code === 4001 || e2?.code === 'ACTION_REJECTED') {
          addressError.value = '用户取消了签名'
          return
        }
        addressError.value = '钱包签名失败：' + (e2?.message || String(e2))
        return
      }
    }

    // 4) POST to /auth/login
    const loginRes = await apiPost('/auth/login', { address: v, message, signature })
    const loginBody = await loginRes.json().catch(() => ({}))
    if (!loginRes.ok) {
      addressError.value = loginBody.error || '登录失败'
      return
    }

    // success
    const token = loginBody.token
    if (token) setToken(token)
    loggedIn.value = true
    loggedInUser.value = loginBody.user?.eth_address || v
    addressError.value = ''
    loadInventory()
    wsConnect()
  } catch (err: any) {
    addressError.value = err?.message || String(err)
  }
}

async function loadInventory() {
  try {
    const res = await apiGet('/rewards')
    if (res.ok) {
      const body = await res.json()
      const inv = body.inventory || {}
      knife.value = inv.knife || 0
      pistol.value = inv.pistol || 0
      bomb.value = inv.bomb || 0
    }
  } catch (err) {
    log.warn('load inventory failed:', err)
  }
}

function logout() {
  wsDisconnect()
  clearToken()
  loggedIn.value = false
  loggedInUser.value = ''
  ethAddress.value = ''
}

// wallet provider detection — tries Phantom first if available
function getProvider(): any {
  const w = window as any
  if (w.phantom?.ethereum?.isPhantom) return w.phantom.ethereum
  return w.ethereum || null
}

// --- ABI encoding helpers ---

function encodeGetPair(tokenA: string, tokenB: string): string {
  // getPair(address,address) selector: 0xe6a43905
  const a = tokenA.toLowerCase().replace('0x', '').padStart(64, '0')
  const b = tokenB.toLowerCase().replace('0x', '').padStart(64, '0')
  return '0xe6a43905' + a + b
}

function encodeSwapExactETHForTokens(tokenA: string, amountOutMin: bigint): string {
  // swapExactETHForTokens(address,uint256) selector: 0xb79c48e5
  const addr = tokenA.toLowerCase().replace('0x', '').padStart(64, '0')
  const amt = amountOutMin.toString(16).padStart(64, '0')
  return '0xb79c48e5' + addr + amt
}

// Uniswap V2 constant product: given ETH in, compute GMTK out
function getAmountOut(ethInWei: bigint, reserveWeth: bigint, reserveGmtk: bigint): bigint {
  const amountInWithFee = ethInWei * 997n
  const numerator = amountInWithFee * reserveGmtk
  const denominator = reserveWeth * 1000n + amountInWithFee
  return numerator / denominator
}

// --- Buy GMTK tokens via Uniswap ---

async function openBuyModal() {
  buyEthAmount.value = '0.01'
  buyEstimatedGmtk.value = ''
  buyError.value = ''
  buyModal.value = true
  await fetchPoolInfo()
}

async function fetchPoolInfo() {
  try {
    gameTokenAddress.value = contracts.game_token

    const eth: any = getProvider()
    if (!eth || !gameTokenAddress.value) return

    // get pair address from factory
    const pairData = encodeGetPair(gameTokenAddress.value, contracts.weth)
    log.info('Pool pair data:', pairData, "gameToken:", gameTokenAddress.value, "WETH:", contracts.weth)
    const pairHex: string = await eth.request({
      method: 'eth_call',
      params: [{ to: contracts.uniswap_v2_factory, data: pairData }, 'latest']
    })

    // address is last 40 bytes of the 32-byte padded return
    const pairAddr = '0x' + pairHex.slice(26)
    if (pairAddr === '0x0000000000000000000000000000000000000000') {
      log.warn('Pool does not exist yet, pairHex:', pairHex)
      return
    }
    log.info('Pool Pair address:', pairAddr)

    // get reserves
    const reservesHex: string = await eth.request({
      method: 'eth_call',
      params: [{ to: pairAddr, data: '0x0902f1ac' }, 'latest']
    })
    // getReserves returns (uint112 reserve0, uint112 reserve1, uint32 blockTimestamp)
    const r0 = BigInt('0x' + reservesHex.slice(2, 66))
    const r1 = BigInt('0x' + reservesHex.slice(66, 130))

    // determine which reserve is WETH / GMTK based on address ordering
    const gmtkIsToken0 = contracts.game_token.toLowerCase() < contracts.weth.toLowerCase()
    const reserveWeth = gmtkIsToken0 ? r1 : r0
    const reserveGmtk = gmtkIsToken0 ? r0 : r1
    poolReserveWeth.value = reserveWeth.toString()
    poolReserveGmtk.value = reserveGmtk.toString()

    log.info('Pool reserves — WETH:', reserveWeth.toString(), 'GMTK:', reserveGmtk.toString())
    // update estimate
    if (Number(buyEthAmount.value) > 0) {
      const ethW = BigInt(Math.floor(Number(buyEthAmount.value) * 1e18))
      const out = getAmountOut(ethW, reserveWeth, reserveGmtk)
      buyEstimatedGmtk.value = out.toString()

      log.info('confirmBuy estimated GMTK:', out.toString())
    }
  } catch (e) {
    log.warn('fetch pool info failed:', e)
  }
}

function estimateGmtk(ethAmount: string, reserveWethStr: string, reserveGmtkStr: string): string {
  if (!ethAmount || Number(ethAmount) <= 0) return '0'
  const reserveWeth = BigInt(reserveWethStr || '0')
  const reserveGmtk = BigInt(reserveGmtkStr || '0')
  if (reserveWeth <= 0n || reserveGmtk <= 0n) return '0'
  try {
    const ethWei = BigInt(Math.floor(Number(ethAmount) * 1e18))
    return getAmountOut(ethWei, reserveWeth, reserveGmtk).toString()
  } catch {
    return '0'
  }
}

async function confirmBuy() {
  const eth: any = getProvider()
  if (!eth) {
    buyError.value = 'No wallet detected'
    return
  }
  buyLoading.value = true
  buyError.value = ''
  try {
    // verify Sepolia network
    const chainId: string = await eth.request({ method: 'eth_chainId' })
    if (chainId !== '0xaa36a7') {
      buyError.value = 'Please switch to Sepolia network (chain ID 11155111)'
      return
    }
    log.info('confirmBuy Sepolia network verified, chainId:', chainId)
    // request accounts
    const accounts: string[] = await eth.request({ method: 'eth_requestAccounts' })
    const from = accounts[0]
    log.info('confirmBuy From account:', from)
    // convert ETH to wei
    const ethWei = BigInt(Math.floor(Number(buyEthAmount.value) * 1e18))
    const hexValue = '0x' + ethWei.toString(16)

    // 5% slippage
    const reserveWeth = BigInt(poolReserveWeth.value)
    const reserveGmtk = BigInt(poolReserveGmtk.value)
    const expectedOut = getAmountOut(ethWei, reserveWeth, reserveGmtk)
    const amountOutMin = expectedOut * 95n / 100n

    log.info('confirmBuy weth:', reserveWeth.toString(), 'gmtk:', reserveGmtk.toString())
    log.info(`confirmBuy Swap: ETH ${buyEthAmount.value} -> GMTK min ${amountOutMin}, expected ${expectedOut}, gameTokenAddress: ${gameTokenAddress.value}`)

    // call swapExactETHForTokens on LiquiditySetup
    const data = encodeSwapExactETHForTokens(gameTokenAddress.value, amountOutMin)
    log.info('confirmBuy LiquiditySetup:', contracts.uniswap_liquidity_setup, 'data:', data, 'gameToken:', gameTokenAddress.value, 'amountOutMin:', amountOutMin.toString())
    const txHash: string = await eth.request({
      method: 'eth_sendTransaction',
      params: [{ from, to: contracts.uniswap_liquidity_setup, value: hexValue, data }]
    })
    log.info(`confirmBuy Transaction hash: ${txHash} from: ${from}, to: ${contracts.uniswap_liquidity_setup}, value: ${hexValue}`)
    // wait for receipt
    await waitForTxReceipt(eth, txHash)
    log.info('confirmBuy Transaction hash:', txHash, 'confirmed')
    addToLog(`Buy GMTK: swapped ${buyEthAmount.value} ETH, tx=${txHash.slice(0, 10)}...`)
    // record on backend
    const recordRes = await apiPost('/tokens/purchase', {
      tx_hash: txHash,
      eth_amount: ethWei.toString(),
      token_amount: expectedOut.toString(),
      rate: '0',
    })
    log.info('confirmBuy Record purchase in server:', recordRes)
    if (!recordRes.ok) {
      log.warn('record purchase failed:', await recordRes.text().catch(() => ''))
    }
    // refresh reserves
    await fetchPoolInfo()
    buyModal.value = false
  } catch (e: any) {
    log.error('confirmBuy failed:', e)
    if (e?.code === 4001 || e?.code === 'ACTION_REJECTED') {
      buyError.value = 'Transaction cancelled'
    } else {
      buyError.value = e?.message || String(e)
    }
  } finally {
    buyLoading.value = false
  }
}

async function waitForTxReceipt(eth: any, txHash: string, timeoutMs = 120000): Promise<any> {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    log.info(`Waiting for transaction receipt: ${txHash}`)
    const receipt = await eth.request({ method: 'eth_getTransactionReceipt', params: [txHash] })
    if (receipt) {
      log.info(`Transaction receipt: ${JSON.stringify(receipt)}`)
      return receipt
    }
    await new Promise(r => setTimeout(r, 2000))
  }
  throw new Error('Transaction receipt timeout')
}

async function fetchTokenBalance(address: string) {
  try {
    const eth = getProvider()
    if (!eth) return
    const addr = address.toLowerCase().replace('0x', '').padStart(64, '0')
    log.info('fetchTokenBalance addr:', addr)
    const hex: string = await eth.request({
      method: 'eth_call',
      params: [{ to: contracts.game_token, data: '0x70a08231' + addr }, 'latest']
    })
    const raw = BigInt(hex)
    const intPart = raw / (10n ** 18n)
    const fracPart = raw % (10n ** 18n)
    const fracStr = fracPart.toString().padStart(18, '0').replace(/0+$/, '')
    profileTokenBalance.value = fracStr ? intPart.toString() + '.' + fracStr : intPart.toString()
    log.info('fetchTokenBalance balance:', profileTokenBalance.value)
  } catch (e: any) {
    log.error('fetchTokenBalance failed:', e)
    profileTokenBalance.value = ''
  }
}

async function queryProfile() {
  const v = queryAddress.value.trim()
  if (!v || !/^0x[a-fA-F0-9]{40}$/.test(v)) {
    profileError.value = 'Invalid address'
    return
  }
  profileError.value = ''
  await fetchAndShowProfile(v)
}

async function queryMyProfile() {
  if (!isLoggedIn()) return
  profileError.value = ''
  try {
    const res = await apiGet('/profile')
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      profileError.value = body.error || 'Failed to load profile'
      return
    }
    const body = await res.json()
    profileUser.value = body.user
    profileOwned.value = body.owned_items || []
    profileListed.value = body.listed_items || []
    profilePlayerItems.value = body.player_items || []
    profileTokenBalance.value = ''
    profileModal.value = true
    fetchTokenBalance(body.user.eth_address)
  } catch (e: any) {
    profileError.value = e?.message || String(e)
  }
}

async function fetchAndShowProfile(address: string) {
  try {
    const res = await apiGet(`/users/${address}`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      profileError.value = body.error || 'User not found'
      return
    }
    const body = await res.json()
    profileUser.value = body.user
    profileOwned.value = body.owned_items || []
    profileListed.value = body.listed_items || []
    profilePlayerItems.value = body.player_items || []
    profileTokenBalance.value = ''
    profileModal.value = true
    fetchTokenBalance(address)
  } catch (e: any) {
    profileError.value = e?.message || String(e)
  }
}

// --- Marketplace ---

function encodeApprove(spender: string, amount: bigint): string {
  const sp = spender.toLowerCase().replace('0x', '').padStart(64, '0')
  const amt = amount.toString(16).padStart(64, '0')
  return '0x095ea7b3' + sp + amt
}

function encodeBuyERC1155(nftContract: string, tokenId: bigint, amount: bigint): string {
  const ctr = nftContract.toLowerCase().replace('0x', '').padStart(64, '0')
  const tid = tokenId.toString(16).padStart(64, '0')
  const amt = amount.toString(16).padStart(64, '0')
  return '0xe318ea1d' + ctr + tid + amt
}

async function loadMarketplace(page: number = 1) {
  marketLoading.value = true
  marketError.value = ''
  try {
    const res = await apiGet('/items', { page: String(page), size: String(PAGE_SIZE) })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      marketError.value = body.error || 'Failed to load marketplace'
      return
    }
    const body = await res.json()
    marketItems.value = body.items || []
    marketTotal.value = body.total || 0
    marketPage.value = page
  } catch (e: any) {
    marketError.value = e?.message || String(e)
  } finally {
    marketLoading.value = false
  }
}

async function switchToMarket() {
  viewMode.value = 'market'
  await loadMarketplace(1)
}

function switchToGame() {
  viewMode.value = 'game'
}

function refreshMarket() {
  loadMarketplace(marketPage.value)
}

async function buyMarketItem(item: any) {
  const eth = getProvider()
  if (!eth) {
    buyItemError.value = 'No wallet detected'
    return
  }
  buyItemId.value = item.id
  buyItemLoading.value = true
  buyItemError.value = ''
  try {
    const chainId: string = await eth.request({ method: 'eth_chainId' })
    if (chainId !== '0xaa36a7') {
      buyItemError.value = 'Please switch to Sepolia network'
      return
    }
    const accounts: string[] = await eth.request({ method: 'eth_requestAccounts' })
    const from = accounts[0]
    const priceWei = parsePrice(item.list_price)
    const amount = BigInt(item.amount || 1)
    const totalPrice = priceWei * amount
    const nftContract = item.nft_contract_address
    const tokenId = BigInt(item.token_id)

    // Step 1: check allowance and approve if needed
    const allowanceData = '0xdd62ed3e' +
      from.toLowerCase().replace('0x', '').padStart(64, '0') +
      contracts.marketplace.toLowerCase().replace('0x', '').padStart(64, '0')
    const allowanceHex: string = await eth.request({
      method: 'eth_call',
      params: [{ to: contracts.game_token, data: allowanceData }, 'latest']
    })
    const allowance = BigInt(allowanceHex)
    if (allowance < totalPrice) {
      log.info('Approving GameToken spend:', totalPrice.toString())
      const approveData = encodeApprove(contracts.marketplace, totalPrice)
      const approveTx: string = await eth.request({
        method: 'eth_sendTransaction',
        params: [{ from, to: contracts.game_token, data: approveData }]
      })
      await waitForTxReceipt(eth, approveTx)
      log.info('Approve confirmed:', approveTx)
    }

    // Step 2: call buyERC1155
    const buyData = encodeBuyERC1155(nftContract, tokenId, amount)
    log.info('Buying item:', item.id, 'nftContract:', nftContract, 'tokenId:', tokenId.toString(), 'amount:', amount.toString())
    const buyTx: string = await eth.request({
      method: 'eth_sendTransaction',
      params: [{ from, to: contracts.marketplace, data: buyData }]
    })
    await waitForTxReceipt(eth, buyTx)
    log.info('Buy confirmed:', buyTx)

    addToLog(`Buy item from market: #${item.id} x${amount} at ${priceWei}, tx=${buyTx.slice(0, 10)}...`)
    // Step 3: notify backend
    const recordRes = await apiPost('/items/' + item.id + '/buy', { tx_hash: buyTx })
    if (!recordRes.ok) {
      log.warn('Backend buy record failed:', await recordRes.text().catch(() => ''))
    }

    // refresh marketplace
    await loadMarketplace(marketPage.value)
  } catch (e: any) {
    log.error('buyMarketItem failed:', e)
    if (e?.code === 4001 || e?.code === 'ACTION_REJECTED') {
      buyItemError.value = 'Transaction cancelled'
    } else {
      buyItemError.value = e?.message || String(e)
    }
  } finally {
    buyItemLoading.value = false
    buyItemId.value = null
  }
}

async function syncToChain(item: any) {
  const eth = getProvider()
  if (!eth) return
  log.info('Syncing item to chain:', item.id)
  try {
    const accounts: string[] = await eth.request({ method: 'eth_requestAccounts' })
    const from = accounts[0]

    // step 1: setApprovalForAll
    const approvalData = '0xa22cb465' +
      contracts.marketplace.toLowerCase().replace('0x', '').padStart(64, '0') +
      '0000000000000000000000000000000000000000000000000000000000000001'
    const approveTx: string = await eth.request({
      method: 'eth_sendTransaction',
      params: [{ from, to: item.nft_contract_address, data: approvalData }]
    })
    await waitForTxReceipt(eth, approveTx)
    log.info('setApprovalForAll confirmed:', approveTx)

    // step 2: listERC1155
    const listData = '0xe8d9b52f' +
      item.nft_contract_address.toLowerCase().replace('0x', '').padStart(64, '0') +
      BigInt(item.token_id).toString(16).padStart(64, '0') +
      BigInt(item.amount || 1).toString(16).padStart(64, '0') +
      parsePrice(item.list_price).toString(16).padStart(64, '0')
    const listTx: string = await eth.request({
      method: 'eth_sendTransaction',
      params: [{ from, to: contracts.marketplace, data: listData }]
    })
    await waitForTxReceipt(eth, listTx)
    log.info('listERC1155 confirmed:', listTx)
    addToLog(`Sync to chain: item #${item.id} listed, tx=${listTx.slice(0, 10)}...`)
  } catch (e: any) {
    log.error('syncToChain failed:', e?.message || e)
  }
}

// update estimate when ETH amount changes
watch(buyEthAmount, () => {
  buyEstimatedGmtk.value = estimateGmtk(buyEthAmount.value, poolReserveWeth.value, poolReserveGmtk.value)
})

</script>

<template>
  <!-- Toast notification -->
  <div v-if="toastMessage" class="toast" :class="toastType">
    {{ toastMessage }}
  </div>

  <main style="display:flex;gap:2px;align-items:flex-start;padding:1rem;width:100%;box-sizing:border-box;flex-wrap:wrap;overflow-x:hidden;">
    <!-- Game view -->
    <div v-if="viewMode === 'game'" style="flex:1;min-width:280px;">
      <TetrisGame ref="tetrisRef" @score="onScoreUpdate" @reward="onReward" />
    </div>

    <!-- Marketplace view -->
    <div v-else style="flex:1;min-width:280px;background:#fff;border-radius:8px;padding:16px;min-height:60vh;">
      <h3 style="margin:0 0 16px 0;color:#333;">Equipment Marketplace</h3>
      <div v-if="marketLoading" style="color:#888;text-align:center;padding:40px;">Loading...</div>
      <div v-else-if="marketError" style="color:#e74c3c;text-align:center;padding:40px;">{{ marketError }}</div>
      <div v-else-if="marketItems.length === 0" style="color:#888;text-align:center;padding:40px;">No items listed for sale.</div>
      <div v-else style="display:flex;flex-direction:column;gap:8px;">
        <div v-for="it in marketItems" :key="it.id"
          style="display:flex;align-items:center;justify-content:space-between;padding:12px;background:#f8f9fa;border-radius:6px;border:1px solid #e0e0e0;gap:12px;flex-wrap:wrap;">
          <div style="flex:1;min-width:120px;">
            <div style="font-weight:600;color:#333;text-transform:capitalize;">{{ it.name || 'Item #' + it.token_id }}</div>
            <div style="font-size:11px;color:#888;">Seller: {{ (it.owner_address || '').slice(0, 10) }}...</div>
          </div>
          <div style="text-align:right;">
            <div style="font-size:10px;color:#888;">Amount: {{ it.amount || 1 }}</div>
            <div style="font-weight:600;color:#2a5c8a;">
              {{ (parsePrice(it.list_price) / (10n ** 18n)).toString() }}.{{ (parsePrice(it.list_price) % (10n ** 18n)).toString().padStart(18,'0').replace(/0+$/, '') || '0' }} GMTK
            </div>
          </div>
          <button
            v-if="isLoggedIn()"
            @click="buyMarketItem(it)"
            :disabled="buyItemLoading && buyItemId === it.id"
            style="padding:6px 16px;background:#4a7c2e;color:#fff;border:none;border-radius:4px;cursor:pointer;white-space:nowrap;">
            {{ buyItemLoading && buyItemId === it.id ? 'Buying...' : 'Buy' }}
          </button>
          <button v-else disabled style="padding:6px 16px;background:#aaa;color:#fff;border:none;border-radius:4px;white-space:nowrap;">
            Login to buy
          </button>
        </div>
        <!-- pagination -->
        <div v-if="marketTotal > PAGE_SIZE" style="display:flex;gap:8px;justify-content:center;margin-top:12px;">
          <button @click="loadMarketplace(marketPage - 1)" :disabled="marketPage <= 1" style="padding:4px 12px;">Prev</button>
          <span style="font-size:12px;color:#666;line-height:2;">Page {{ marketPage }}</span>
          <button @click="loadMarketplace(marketPage + 1)" :disabled="marketPage * PAGE_SIZE >= marketTotal" style="padding:4px 12px;">Next</button>
        </div>
      </div>
      <div v-if="buyItemError" style="color:#e74c3c;font-size:12px;margin-top:8px;">{{ buyItemError }}</div>
    </div>

    <div style="width:160px;flex:0 0 160px;display:flex;flex-direction:column;gap:8px;">

      <!-- View toggle -->
      <div style="display:flex;gap:4px;">
        <button @click="switchToGame" :style="{flex:1,padding:'6px 4px',fontSize:'11px',borderRadius:'4px',cursor:'pointer',background:viewMode==='game'?'#4a7c2e':'#444',color:'#fff',border:'none'}">Game</button>
        <button @click="switchToMarket" :style="{flex:1,padding:'6px 4px',fontSize:'11px',borderRadius:'4px',cursor:'pointer',background:viewMode==='market'?'#4a7c2e':'#444',color:'#fff',border:'none'}">Market</button>
      </div>

      <!-- Login section (hidden in market view) -->
      <template v-if="viewMode === 'game'">
        <div v-if="!loggedIn" style="display:flex;flex-direction:column;gap:6px;">
          <div style="display:flex;gap:6px;align-items:center;">
            <input v-model="ethAddress" placeholder="Ethereum address (0x...)" style="padding:6px;border-radius:4px;border:1px solid #444;flex:1;min-width:0;box-sizing:border-box;background:#1a1a1a;color:#fff" />
            <button @click="login" style="padding:6px;flex:0 0 auto;white-space:nowrap">Login</button>
          </div>
          <div v-if="addressError" style="color:#ff6666;font-size:12px;margin-top:4px;word-break:break-all">{{ addressError }}</div>
        </div>

        <div v-else style="display:flex;flex-direction:column;gap:6px;padding:6px 0;">
          <div style="display:flex;justify-content:space-between;align-items:center;">
            <span style="font-size:12px;color:#888;">已登录</span>
            <button @click="logout" style="padding:2px 6px;font-size:11px;background:transparent;border:1px solid #555;color:#ccc;border-radius:4px;cursor:pointer">Logout</button>
          </div>
          <div style="font-size:11px;color:#aaa;word-break:break-all;background:#1a1a1a;padding:4px 6px;border-radius:4px;">
            {{ loggedInUser }}
          </div>
        </div>

        <div v-if="!loggedIn" style="display:flex;flex-direction:column;gap:4px;">
          <div style="display:flex;gap:4px;">
            <input v-model="queryAddress" placeholder="0x..." style="padding:4px 6px;border-radius:4px;border:1px solid #444;flex:1;min-width:0;box-sizing:border-box;background:#1a1a1a;color:#fff;font-size:10px;" />
            <button @click="queryProfile" style="padding:4px 8px;font-size:10px;flex:0 0 auto;white-space:nowrap;">Query</button>
          </div>
          <div v-if="profileError" style="color:#ff6666;font-size:10px;">{{ profileError }}</div>
        </div>
        <div v-else style="display:flex;flex-direction:column;gap:4px;">
          <button @click="queryMyProfile" style="padding:4px 8px;font-size:11px;width:100%;">My Profile</button>
          <div v-if="profileError" style="color:#ff6666;font-size:10px;">{{ profileError }}</div>
        </div>

        <div style="display:flex;gap:8px;justify-content:center;width:100%;">
          <button @click="startGame">Start</button>
          <button @click="pauseGame">Pause</button>
        </div>

        <div v-if="loggedIn" style="display:flex;justify-content:center;width:100%;">
          <button @click="openBuyModal" style="padding:6px 12px;background:#2a5c8a;color:#fff;border:1px solid #2a5c8a;border-radius:4px;cursor:pointer;width:100%;">Buy GMTK</button>
        </div>

        <div style="flex:1;display:flex;align-items:stretch;">
          <ScoreBoard :score="score" :knife="knife" :pistol="pistol" :bomb="bomb" @use="onUse" @sell="onSell" style="flex:1;" />
        </div>

        <div class="activity-log-panel">
          <div class="activity-log-header">
            <span>Activity Log</span>
            <button @click="activityLog = []" class="activity-log-clear">Clear</button>
          </div>
          <textarea
            readonly
            :value="activityLog.join('\n')"
            placeholder="Key on-chain operations will appear here..."
            class="activity-log-area"
          ></textarea>
        </div>
      </template>

      <!-- Market sidebar: GMTK balance + quick buy GMTK -->
      <template v-else>
        <div v-if="!loggedIn" style="display:flex;flex-direction:column;gap:6px;">
          <div style="display:flex;gap:6px;align-items:center;">
            <input v-model="ethAddress" placeholder="Ethereum address (0x...)" style="padding:6px;border-radius:4px;border:1px solid #444;flex:1;min-width:0;box-sizing:border-box;background:#1a1a1a;color:#fff;font-size:10px;" />
            <button @click="login" style="padding:6px;flex:0 0 auto;white-space:nowrap;font-size:11px;">Login</button>
          </div>
          <div v-if="addressError" style="color:#ff6666;font-size:10px;">{{ addressError }}</div>
        </div>
        <div v-else style="display:flex;flex-direction:column;gap:6px;">
          <div style="font-size:10px;color:#888;">Logged in</div>
          <div style="font-size:10px;color:#aaa;word-break:break-all;background:#1a1a1a;padding:4px 6px;border-radius:4px;">{{ loggedInUser }}</div>
          <button @click="logout" style="padding:2px 6px;font-size:10px;background:transparent;border:1px solid #555;color:#ccc;border-radius:4px;cursor:pointer;">Logout</button>
          <button @click="openBuyModal" style="padding:6px 8px;font-size:10px;background:#2a5c8a;color:#fff;border:none;border-radius:4px;cursor:pointer;">Buy GMTK</button>
        </div>
        <button @click="refreshMarket" style="padding:4px 8px;font-size:10px;width:100%;background:#555;color:#fff;border:none;border-radius:4px;cursor:pointer;margin-top:8px;">Refresh</button>
      </template>
    </div>
  </main>

  <!-- Buy modal -->
  <div v-if="buyModal" class="modal-overlay" @click.self="buyModal = false">
    <div class="modal">
      <h3>Buy GMTK Tokens</h3>
      <div class="modal-field">
        <label>ETH to spend</label>
        <input type="number" v-model="buyEthAmount" step="0.001" min="0" placeholder="0.01" />
      </div>
      <div class="modal-info">
        Estimated GMTK: {{ buyEstimatedGmtk || '...' }}
      </div>
      <div class="modal-info" style="font-size:10px;color:#666;">
        Pool: {{ poolReserveGmtk !== '0' ? (Number(BigInt(poolReserveGmtk) * 10000n / BigInt(poolReserveWeth)) / 10000).toFixed(4) : '...' }} GMTK per ETH
      </div>
      <div v-if="buyError" class="modal-info" style="color:#ff6666;">
        {{ buyError }}
      </div>
      <div class="modal-actions">
        <button @click="confirmBuy" :disabled="buyLoading || Number(buyEthAmount) <= 0">
          {{ buyLoading ? 'Confirming...' : 'Buy' }}
        </button>
        <button class="cancel" @click="buyModal = false">Cancel</button>
      </div>
    </div>
  </div>

  <!-- Sell modal -->
  <div v-if="sellModal" class="modal-overlay" @click.self="cancelSell">
    <div class="modal">
      <h3>Sell {{ sellItem }}</h3>
      <div class="modal-field">
        <label>Amount (max {{ sellMax }})</label>
        <input type="number" v-model.number="sellAmount" :min="1" :max="sellMax" />
      </div>
      <div class="modal-field">
        <label>Unit price (in tokenG smallest unit)</label>
        <input type="text" v-model="sellPrice" placeholder="1000000000000000000" />
      </div>
      <div class="modal-info">
        Total: {{ sellAmount }} × {{ sellPrice }} = {{ sellAmount * Number(sellPrice) || 0 }}
      </div>
      <div class="modal-actions">
        <button @click="confirmSell" :disabled="sellAmount <= 0 || sellAmount > sellMax">Confirm</button>
        <button class="cancel" @click="cancelSell">Cancel</button>
      </div>
    </div>
  </div>

  <!-- Profile modal -->
  <div v-if="profileModal" class="modal-overlay" @click.self="profileModal = false">
    <div class="modal" style="max-width:500px;">
      <h3>User Profile</h3>
      <div v-if="profileUser" style="display:flex;flex-direction:column;gap:8px;">
        <div class="modal-field">
          <label>Address</label>
          <div style="font-size:11px;color:#aaa;word-break:break-all;background:#1a1a1a;padding:4px 6px;border-radius:4px;">{{ profileUser.eth_address }}</div>
        </div>
        <div class="modal-field">
          <label>Username</label>
          <div style="font-size:12px;">{{ profileUser.username || '-' }}</div>
        </div>
        <div class="modal-field">
          <label>GMTK Balance</label>
          <div style="font-size:12px;color:#4fc3f7;">{{ profileTokenBalance || '...' }} GMTK</div>
        </div>
        <div style="display:flex;gap:12px;">
          <span style="font-size:12px;color:#aaa;">Owned: <b>{{ profileOwned.length }}</b></span>
          <span style="font-size:12px;color:#aaa;">Listed: <b>{{ profileListed.length }}</b></span>
          <span style="font-size:12px;color:#aaa;">Rewards: <b>{{ profilePlayerItems.length }}</b></span>
        </div>
        <div v-if="profilePlayerItems.length > 0" class="modal-field">
          <label>Game Rewards</label>
          <div style="display:flex;gap:8px;">
            <span v-for="pi in profilePlayerItems" :key="pi.item_type" style="font-size:10px;padding:2px 6px;background:#1a3a1a;border-radius:3px;">{{ pi.item_type }} × {{ pi.quantity }}</span>
          </div>
        </div>
        <div v-if="profileOwned.length > 0" class="modal-field">
          <label>Owned Items</label>
          <div style="max-height:120px;overflow-y:auto;">
            <div v-for="it in profileOwned" :key="it.id" style="font-size:10px;padding:2px 4px;margin:2px 0;background:#1a1a1a;border-radius:3px;display:flex;justify-content:space-between;gap:4px;">
              <span>{{ it.name || '#' + it.token_id }}</span>
              <span style="color:#888;">{{ it.token_standard }} {{ it.is_listed ? '(listed)' : '' }}</span>
            </div>
          </div>
        </div>
        <div v-if="profileListed.length > 0" class="modal-field">
          <label>Listed Items</label>
          <div style="max-height:120px;overflow-y:auto;">
            <div v-for="it in profileListed" :key="'l'+it.id" style="font-size:10px;padding:2px 4px;margin:2px 0;background:#1a1a1a;border-radius:3px;display:flex;justify-content:space-between;gap:4px;align-items:center;">
              <span>{{ it.name || '#' + it.token_id }}</span>
              <span style="color:#4a7c2e;">Price: {{ it.list_price }}</span>
              <button @click="syncToChain(it)" style="padding:1px 6px;font-size:9px;background:#2a5c8a;color:#fff;border:none;border-radius:3px;cursor:pointer;">Sync</button>
            </div>
          </div>
        </div>
        <div v-if="profileOwned.length === 0 && profileListed.length === 0 && profilePlayerItems.length === 0" style="font-size:11px;color:#666;">
          No items.
        </div>
      </div>
      <div class="modal-actions" style="margin-top:12px;">
        <button class="cancel" @click="profileModal = false">Close</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.toast {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 2000;
  padding: 14px 32px;
  border-radius: 8px;
  font-size: 18px;
  font-weight: 700;
  color: #fff;
  animation: toast-in 0.3s ease;
}
.toast.success { background: #4a7c2e; }
.toast.error   { background: #c0392b; }

@keyframes toast-in {
  from { opacity: 0; transform: translateX(-50%) translateY(-12px); }
  to   { opacity: 1; transform: translateX(-50%) translateY(0); }
}

header {
  line-height: 1.5;
}

.logo {
  display: block;
  margin: 0 auto 2rem;
}

@media (min-width: 1024px) {
  header {
    display: flex;
    place-items: center;
    padding-right: calc(var(--section-gap) / 2);
  }

  .logo {
    margin: 0 2rem 0 0;
  }

  header .wrapper {
    display: flex;
    place-items: flex-start;
    flex-wrap: wrap;
  }
}

.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(0,0,0,0.6);
  display: flex; align-items: center; justify-content: center;
  z-index: 1000;
}
.modal {
  background: #222; color: #fff; border-radius: 8px;
  padding: 20px; min-width: 300px; max-width: 90vw;
}
.modal h3 { margin: 0 0 12px 0; text-transform: capitalize; }
.modal-field { margin-bottom: 10px; }
.modal-field label { display: block; font-size: 12px; color: #aaa; margin-bottom: 4px; }
.modal-field input {
  width: 100%; padding: 6px 8px; border-radius: 4px;
  border: 1px solid #555; background: #1a1a1a; color: #fff;
  box-sizing: border-box;
}
.modal-info { font-size: 12px; color: #aaa; margin-bottom: 14px; word-break: break-all; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; }
.modal-actions button {
  padding: 6px 16px; border-radius: 4px; border: 1px solid #555;
  background: #4a7c2e; color: #fff; cursor: pointer;
}
.modal-actions button.cancel { background: #555; }
.modal-actions button:disabled { opacity: 0.4; cursor: default; }

.activity-log-panel {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
.activity-log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}
.activity-log-header span {
  font-size: 11px;
  color: #333;
  font-weight: 600;
}
.activity-log-clear {
  padding: 1px 8px;
  font-size: 10px;
  background: transparent;
  border: 1px solid #ccc;
  color: #666;
  border-radius: 3px;
  cursor: pointer;
}
.activity-log-area {
  width: 100%;
  height: 160px;
  padding: 6px 8px;
  border-radius: 4px;
  border: 1px solid #ccc;
  background: #fff;
  color: #000;
  font-family: 'Courier New', Courier, monospace;
  font-size: 10px;
  line-height: 1.5;
  resize: vertical;
  box-sizing: border-box;
  white-space: pre;
  overflow-wrap: normal;
  overflow-x: auto;
}
.activity-log-area::placeholder {
  color: #999;
}
</style>
