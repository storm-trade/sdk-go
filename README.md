# Storm Trade Go SDK

Go SDK for [Storm Trade](https://storm.tg) — perpetual futures on TON blockchain.

Supports both **v3** (Smart Account, intent-based) and **v2** (direct wallet transactions) protocols.

## Installation

```bash
go get github.com/storm-trade/sdk-go
```

## Architecture

```
config/                     Shared types, config API (Network, Market, Asset, Direction)
storm/                      v3 high-level client (Smart Account + sequencer)
stormv2/                    v2 high-level client (direct wallet transactions)
sequencer/                  v3 sequencer REST API client
oracle/                     Oracle price API client (for v2 margin operations)
matcher/                    v2 matcher broadcast client
client/
├── vamm/                   On-chain vAMM queries
├── vault/                  On-chain Vault queries
├── smartaccount/           On-chain Smart Account operations
└── positionmanager/        On-chain Position Manager queries
contracts/                  Data types and TLB schemas
tlb/                        Order types, position state, error codes
```

## v3 SDK (Smart Account)

v3 uses signed intents sent to the sequencer. The sequencer handles oracle data, bundling, and execution.

### Quick Start

```go
ctx := context.Background()

pool := liteclient.NewConnectionPool()
pool.AddConnectionsFromConfigUrl(ctx, "https://ton-blockchain.github.io/testnet-global.config.json")
api := ton.NewAPIClient(pool, ton.ProofCheckPolicyUnsafe).WithRetry(10)

words := strings.Split("your seed phrase ...", " ")
w, _ := wallet.FromSeed(api, words, wallet.V4R2)

factory := smartaccount.NewFactory(api, address.MustParseAddr(config.Networks[config.Testnet].FactoryAddress))
saAddr, _ := factory.GetSmartAccountAddress(ctx, w.WalletAddress())

privKey := ed25519.PrivateKey(yourKeyBytes)
client := storm.NewClient(config.Testnet,
    storm.WithTONApi(api),
    storm.WithWallet(w),
    storm.WithSmartAccount(saAddr),
    storm.WithSigner(privKey),
    storm.WithClockSkew(5*time.Second),
)

btc, _ := client.Market("BTC", "USDT")
amount := tlb.MustFromDecimal("100", 9)

result, _ := client.PlaceMarketOrder(ctx, btc, config.Long, &amount, 3_000_000_000)
fmt.Printf("Order hash: %x\n", result.OrderHash)
```

### Options

| Option | Purpose |
|--------|---------|
| `WithSigner(key)` | ED25519 private key for signing orders |
| `WithSmartAccount(addr)` | Smart Account address |
| `WithTONApi(api)` | TON API client (for deposits, withdrawals, on-chain queries) |
| `WithWallet(w)` | Wallet for on-chain transactions |
| `WithClockSkew(d)` | Compensate local clock offset (recommended: 5s) |
| `WithGasless()` | Use gasless execution mode |
| `WithInit()` | Initialize Smart Account on first deposit |
| `WithPublicKey(key)` | Set public key during initialization |
| `WithExpiration(uint32)` | Custom order expiration (unix timestamp) |
| `WithQueryID(uint64)` | Custom query ID |

### Trading

All amounts in orders are in **9 decimals** internally. Leverage is also in 9 decimals: 3x = `3_000_000_000`.

```go
amount := tlb.MustFromDecimal("100", 9)

// Market order
result, _ := client.PlaceMarketOrder(ctx, market, config.Long, &amount, 3_000_000_000)

// With SL/TP attached
sl := tlb.MustFromTON("60000")
tp := tlb.MustFromTON("80000")
result, _ := client.PlaceMarketOrder(ctx, market, config.Long, &amount, 3_000_000_000,
    storm.WithStopLoss(&sl), storm.WithTakeProfit(&tp))

// Limit order
limitPrice := tlb.MustFromTON("65000")
result, _ := client.PlaceLimitOrder(ctx, market, config.Long, &amount, 3_000_000_000, &limitPrice)

// Stop-limit order
stopPrice := tlb.MustFromTON("64000")
result, _ := client.PlaceStopLimitOrder(ctx, market, config.Long, &amount, 3_000_000_000, &limitPrice, &stopPrice)
```

### Position Management

```go
// Close position (partial)
size := tlb.MustFromDecimal("0.001", 9)
result, _ := client.ClosePosition(ctx, market, config.Long, &size)

// Close entire position (queries size on-chain)
result, _ := client.ClosePositionFull(ctx, market, config.Long)

// Standalone SL/TP on existing position
trigger := tlb.MustFromTON("60000")
result, _ := client.PlaceStopLoss(ctx, market, config.Long, &size, &trigger)
trigger = tlb.MustFromTON("80000")
result, _ = client.PlaceTakeProfit(ctx, market, config.Long, &size, &trigger)

// Margin
margin := tlb.MustFromDecimal("50", 9)
result, _ := client.AddMargin(ctx, market, config.Long, &margin)
result, _ := client.RemoveMargin(ctx, market, config.Long, &margin)

// Cancel by order hash
client.CancelOrder(ctx, result.OrderHash)
```

### Deposits & Withdrawals

Amounts use the **asset's native decimals** (6 for USDT, 9 for TON/NOT).

```go
amount := tlb.MustFromDecimal("100", 6)
client.Deposit(ctx, "USDT", &amount)

// First deposit — init Smart Account + register key
client.Deposit(ctx, "TON", &amount, storm.WithInit(), storm.WithPublicKey(pubKey))

client.Withdraw(ctx, "USDT", &amount)
```

| init | publicKey | Result |
|------|-----------|--------|
| true | set | Deploy SA + register key |
| true | nil | Deploy SA, no key |
| false | nil | Normal deposit |
| false | set | **Error 115** — contract rejects key without init |

### Result

Every order method returns `*PlaceOrderResult`:

```go
type PlaceOrderResult struct {
    QueryID        uint64
    OrderHash      []byte                        // use for CancelOrder
    StopLossID     uint64
    StopLossHash   []byte
    TakeProfitID   uint64
    TakeProfitHash []byte
    Response       *sequencer.PlaceOrderResponse  // OK, Trace, Intent
}
```

### Key Management

```go
client.AddPublicKey(ctx)
```

---

## v2 SDK (Direct Wallet)

v2 sends TON transactions directly to Vault and Position Manager contracts. Includes executor fee as a second message automatically.

### Quick Start

```go
ctx := context.Background()

pool := liteclient.NewConnectionPool()
pool.AddConnectionsFromConfigUrl(ctx, "https://ton-blockchain.github.io/testnet-global.config.json")
api := ton.NewAPIClient(pool, ton.ProofCheckPolicyUnsafe).WithRetry(10)

words := strings.Split("your seed phrase ...", " ")
w, _ := wallet.FromSeed(api, words, wallet.V4R2)

client := stormv2.NewClient(config.Testnet,
    stormv2.WithWallet(w),
    stormv2.WithTONApi(api),
)

btc, _ := client.Market("BTC", "USDT")
amount := tlb.MustFromDecimal("10", 6)  // collateral decimals (6 for USDT)

result, _ := client.PlaceMarketOrder(ctx, btc, config.Long, &amount, 3_000_000_000)
fmt.Printf("TX hash: %s\n", result.Hash)
```

### Options

| Option | Purpose |
|--------|---------|
| `WithWallet(w)` | Wallet for signing and sending transactions |
| `WithTONApi(api)` | TON API client (required for all operations) |
| `WithReferralID(id)` | Referral NFT index (applied on first order per market) |
| `WithStopLoss(price)` | Attach SL trigger to new order |
| `WithTakeProfit(price)` | Attach TP trigger to new order |
| `WithExpiration(uint32)` | Custom order expiration (unix timestamp) |
| `WithQueryID(uint64)` | Custom query ID for jetton transfers |

### Key Differences from v3

| | v3 (storm/) | v2 (stormv2/) |
|---|---|---|
| Account model | Smart Account (one per trader) | Position Manager (one per market) |
| Order submission | Signed intent → sequencer | Wallet transaction → blockchain |
| Oracle data | Sequencer injects | SDK fetches for margin ops |
| SL/TP on open | OrderRequest intents | Trigger prices in order cell |
| Cancel | By order hash | By orderType + orderIndex |
| Executor fee | Not needed | Auto-appended |
| Amount decimals | 9 (internal) | Collateral decimals (6 USDT, 9 TON) |
| PM init | Not applicable | Auto-detected |

### Trading

Amounts use **collateral decimals** (6 for USDT, 9 for TON). The vault converts internally.

```go
amount := tlb.MustFromDecimal("10", 6)  // 10 USDT

// Market order with SL/TP
sl := tlb.MustFromTON("60000")
tp := tlb.MustFromTON("80000")
result, _ := client.PlaceMarketOrder(ctx, market, config.Long, &amount, 3_000_000_000,
    stormv2.WithStopLoss(&sl), stormv2.WithTakeProfit(&tp))

// Limit order
limitPrice := tlb.MustFromTON("65000")
result, _ := client.PlaceLimitOrder(ctx, market, config.Long, &amount, 3_000_000_000, &limitPrice)

// Stop-limit
stopPrice := tlb.MustFromTON("64000")
result, _ := client.PlaceStopLimitOrder(ctx, market, config.Long, &amount, 3_000_000_000, &limitPrice, &stopPrice)

// With referral
result, _ := client.PlaceMarketOrder(ctx, market, config.Long, &amount, 3_000_000_000,
    stormv2.WithReferralID(123))
```

### Position Management

Close and SL/TP use **9 decimals** (base asset size). Remove margin uses **9 decimals** (internal format).

```go
// Close position
size := tlb.MustFromDecimal("0.001", 9)
result, _ := client.ClosePosition(ctx, market, config.Long, &size)

// Standalone SL/TP
trigger := tlb.MustFromTON("60000")
result, _ := client.PlaceStopLoss(ctx, market, config.Long, &size, &trigger)

// Add margin (collateral decimals)
margin := tlb.MustFromDecimal("5", 6)
result, _ := client.AddMargin(ctx, market, config.Long, &margin)

// Remove margin (9 decimals, internal format)
margin = tlb.MustFromDecimal("2", 9)
result, _ := client.RemoveMargin(ctx, market, config.Long, &margin)

// Cancel by type + index (from orders query)
result, _ := client.CancelOrder(ctx, market, config.Long, 2, 0)  // type=2 (limit), index=0
```

### Order Types for Cancel

| Type | Name |
|------|------|
| 0 | Stop Loss |
| 1 | Take Profit |
| 2 | Limit |
| 3 | Market |

---

## Markets & Assets

Shared between v3 and v2. Fetched from the config API automatically.

```go
btc, _ := client.Market("BTC", "USDT")
eth, _ := client.Market("ETH", "TON")
btc, _ := client.Market("BTC")  // first match
markets, _ := client.Markets()
usdt, _ := client.Asset("USDT")
```

## Market Data (On-Chain)

High-level wrappers available in v3 (`storm/`). For v2, use low-level clients directly.

```go
spotPrice, _ := client.GetSpotPrice(ctx, market)
state, _ := client.GetAmmState(ctx, market)
settings, _ := client.GetExchangeSettings(ctx, market)
oracle, _ := client.GetOracleData(ctx, market)
vaultData, _ := client.GetVaultData(ctx, "USDT")
```

Low-level clients (usable with both v3 and v2):

```go
vc := vamm.NewClient(tonAPI, market.VammAddress)
spotPrice, _ := vc.GetSpotPrice(ctx)

v := vault.NewClient(tonAPI, market.VaultAddress)
vaultData, _ := v.GetVaultData(ctx)
```

## Sequencer API (v3)

```go
seq := sequencer.NewClient(sequencer.TestnetURL)

state, _ := seq.GetAccountState(ctx, saAddress)
balances, _ := seq.GetBalances(ctx, saAddress)
positions, _ := seq.GetPositions(ctx, saAddress)
status, _ := seq.GetStatus(ctx)
intent, _ := seq.GetIntent(ctx, hash)
resp, _ := seq.PlaceOrder(ctx, placeOrderRequest)
depth, _ := seq.GetOrderbookDepth(ctx, assetID, levels)
```

## Oracle API (v2)

Used internally by stormv2 for add/remove margin. Also available directly:

```go
oc := oracle.NewClient(oracle.TestnetURL)
price, _ := oc.GetSignedPrice(ctx, "BTC")
payload := oracle.BuildSimplePayload(price)              // for base markets (USDT)
payload := oracle.BuildSettlementPayload(base, settlement) // for coinm markets (TON, NOT)
```

## CLI Tools

### v3 CLI (`cmd/example`)

```bash
go run ./cmd/example keygen
go run ./cmd/example add-key
go run ./cmd/example markets
go run ./cmd/example balance
go run ./cmd/example positions
go run ./cmd/example info BTC USDT
go run ./cmd/example deposit USDT 100
go run ./cmd/example deposit TON 0.5 --init
go run ./cmd/example withdraw USDT 50
go run ./cmd/example order market BTC USDT long 100 3
go run ./cmd/example order limit BTC USDT long 100 3 --limit=65000
go run ./cmd/example order market BTC USDT long 100 3 --sl=60000 --tp=80000
go run ./cmd/example close BTC USDT long 0.001
go run ./cmd/example close-all BTC USDT long
go run ./cmd/example stop-loss BTC USDT long 0.001 60000
go run ./cmd/example take-profit BTC USDT long 0.001 80000
go run ./cmd/example add-margin BTC USDT long 50
go run ./cmd/example remove-margin BTC USDT long 20
go run ./cmd/example cancel <hash>
```

### v2 CLI (`cmd/v2example`)

```bash
go run ./cmd/v2example markets
go run ./cmd/v2example order market BTC USDT long 10 3
go run ./cmd/v2example order market BTC USDT long 10 3 --sl=60000 --tp=80000
go run ./cmd/v2example order limit BTC USDT long 10 3 --limit=60000
go run ./cmd/v2example order stop-limit BTC USDT long 10 3 --limit=60000 --stop=62000
go run ./cmd/v2example order market BTC USDT long 10 3 --ref=123
go run ./cmd/v2example orders BTC USDT long
go run ./cmd/v2example close BTC USDT long 0.001
go run ./cmd/v2example stop-loss BTC USDT long 0.001 60000
go run ./cmd/v2example take-profit BTC USDT long 0.001 80000
go run ./cmd/v2example add-margin BTC USDT long 5
go run ./cmd/v2example remove-margin BTC USDT long 2
go run ./cmd/v2example cancel BTC USDT long 2 0
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `STORM_SEED` | Wallet seed phrase (24 words) | required |
| `STORM_PRIVATE_KEY` | ED25519 private key, hex (v3 only) | required for v3 orders |
| `STORM_NETWORK` | `testnet` or `mainnet` | `testnet` |
| `STORM_WALLET_VERSION` | `v3r2`, `v4r2`, `v5r1` | `v4r2` |
| `STORM_SUBWALLET_ID` | Custom subwallet ID | version default |
| `STORM_GLOBAL_ID` | Override network global ID (v5r1) | auto from network |

## Contract Addresses

| Network | Factory |
|---------|---------|
| Testnet | `kQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_Qvr` |
| Mainnet | `EQA34l2ywiFdu_kb-HZMqLngFVDjw0DJZHo1aBokOap8xVMU` |

Market and vault addresses are fetched automatically from the config API.

## Error Codes

| Code | Source | Description |
|------|--------|-------------|
| 115 | Smart Account | Key init not allowed without init flag |
| 170 | Smart Account | Invalid `created_at` timestamp |
| 171 | Smart Account | Query already processed |
| 402 | Smart Account | Public key not registered |
| 411 | vAMM | Wrong order size |
| 426 | vAMM | Wrong leverage |
| 471 | vAMM | Position not ready for close (cooldown) |
| 511 | Position Manager | Position not found |
| 514 | Position Manager | Order not found |
| 522 | Position Manager | Position Manager not initialized |
| 526 | Position Manager | Direction mismatch |

## Links

- [Storm Trade](https://storm.tg)
- [Documentation](https://docs.storm.tg)
- [Telegram](https://t.me/StormTradeBot)

## License

MIT
