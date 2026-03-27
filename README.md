# Storm Trade Go SDK

Go SDK for [Storm Trade](https://storm.tg) — perpetual futures on TON blockchain.

## Installation

```bash
go get github.com/storm-trade/sdk-go
```

## Quick Start

```go
ctx := context.Background()

// Connect to TON
pool := liteclient.NewConnectionPool()
pool.AddConnectionsFromConfigUrl(ctx, "https://ton-blockchain.github.io/testnet-global.config.json")
api := ton.NewAPIClient(pool, ton.ProofCheckPolicyUnsafe).WithRetry(10)

// Create wallet from seed phrase
words := strings.Split("your seed phrase ...", " ")
w, _ := wallet.FromSeed(api, words, wallet.V4R2)

// Derive Smart Account address from factory
client := storm.NewClient(storm.Testnet)
factory := smartaccount.NewFactory(api, address.MustParseAddr(client.FactoryAddress()))
saAddr, _ := factory.GetSmartAccountAddress(ctx, w.WalletAddress())

// Create SDK client with all options
privKey := ed25519.PrivateKey(yourKeyBytes)
client = storm.NewClient(storm.Testnet,
    storm.WithTONApi(api),
    storm.WithWallet(w),
    storm.WithSmartAccount(saAddr),
    storm.WithSigner(privKey),
    storm.WithClockSkew(5*time.Second),
)

// Place market order: 100 USDT margin, 3x leverage, long BTC
btc, _ := client.Market("BTC", "USDT")
amount := tlb.MustFromDecimal("100", 9)

result, _ := client.PlaceMarketOrder(ctx, btc, storm.Long, &amount, 3_000_000_000)
fmt.Printf("Order hash: %x\n", result.OrderHash)
```

## Architecture

The SDK has three layers:

```
storm/                      High-level client (recommended)
sequencer/                  Sequencer REST API client
client/
├── vamm/                   On-chain vAMM queries
├── vault/                  On-chain Vault queries
├── smartaccount/           On-chain Smart Account operations
└── positionmanager/        On-chain Position Manager queries
contracts/                  Data types and TLB schemas
tlb/                        Order types, position state, error codes
```

Most users only need the `storm/` package. Lower-level packages are available for custom workflows.

## Client Setup

### Networks

```go
client := storm.NewClient(storm.Testnet, opts...)
client := storm.NewClient(storm.Mainnet, opts...)
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

## Markets & Assets

Markets and assets are fetched from the config API automatically:

```go
// By name + settlement token
btc, _ := client.Market("BTC", "USDT")
eth, _ := client.Market("ETH", "TON")

// First match by name
btc, _ := client.Market("BTC")

// List all markets
markets, _ := client.Markets()

// Assets (for deposits/withdrawals)
usdt, _ := client.Asset("USDT")
```

## Trading

All amounts in orders are in **9 decimals** internally. Use `tlb.MustFromDecimal("100", 9)` for 100 units.

Leverage is also in 9 decimals: 3x = `3_000_000_000`.

### Open Position

```go
amount := tlb.MustFromDecimal("100", 9)

// Market order
result, err := client.PlaceMarketOrder(ctx, market, storm.Long, &amount, 3_000_000_000)

// Limit order
limitPrice := tlb.MustFromTON("65000")
result, err := client.PlaceLimitOrder(ctx, market, storm.Long, &amount, 3_000_000_000, &limitPrice)

// Stop-limit order
stopPrice := tlb.MustFromTON("64000")
result, err := client.PlaceStopLimitOrder(ctx, market, storm.Long, &amount, 3_000_000_000, &limitPrice, &stopPrice)
```

### Open with SL/TP

Attach stop-loss and take-profit to a new order via `OrderRequest` intents:

```go
sl := tlb.MustFromTON("60000")
tp := tlb.MustFromTON("80000")

result, err := client.PlaceMarketOrder(ctx, market, storm.Long, &amount, 3_000_000_000,
    storm.WithStopLoss(&sl),
    storm.WithTakeProfit(&tp),
)

// Result contains hashes for all three orders
fmt.Printf("Order:      %x\n", result.OrderHash)
fmt.Printf("StopLoss:   %x\n", result.StopLossHash)
fmt.Printf("TakeProfit: %x\n", result.TakeProfitHash)
```

### Close Position

Close uses `TakeOrder` with `triggerPrice=0` (market execution). Direction is the **same** as the position, not opposite.

```go
// Partial close (0.001 BTC)
size := tlb.MustFromDecimal("0.001", 9)
result, err := client.ClosePosition(ctx, market, storm.Long, &size)

// Close entire position (queries size on-chain)
result, err := client.ClosePositionFull(ctx, market, storm.Long)
```

### Standalone SL/TP

Set stop-loss or take-profit on an existing position. These are independent orders, not linked to an opening order.

```go
size := tlb.MustFromDecimal("0.001", 9)
trigger := tlb.MustFromTON("60000")

// Stop loss
result, err := client.PlaceStopLoss(ctx, market, storm.Long, &size, &trigger)

// Take profit
trigger = tlb.MustFromTON("80000")
result, err := client.PlaceTakeProfit(ctx, market, storm.Long, &size, &trigger)
```

### Margin Management

```go
margin := tlb.MustFromDecimal("50", 9)

// Add margin to reduce liquidation risk
result, err := client.AddMargin(ctx, market, storm.Long, &margin)

// Remove excess margin
result, err := client.RemoveMargin(ctx, market, storm.Long, &margin)
```

### Cancel Order

```go
err := client.CancelOrder(ctx, result.OrderHash)
```

## Deposits & Withdrawals

Deposit and withdrawal amounts use the **asset's native decimals** (6 for USDT, 9 for TON/NOT).

```go
// Deposit 100 USDT
amount := tlb.MustFromDecimal("100", 6)
err := client.Deposit(ctx, "USDT", &amount)

// Deposit 1 TON
amount = tlb.MustFromDecimal("1", 9)
err = client.Deposit(ctx, "TON", &amount)

// First deposit — initialize Smart Account with public key
pubKey := ed25519.PrivateKey(keyBytes).Public().(ed25519.PublicKey)
err = client.Deposit(ctx, "TON", &amount,
    storm.WithInit(),
    storm.WithPublicKey(pubKey),
)

// Withdraw
amount = tlb.MustFromDecimal("50", 6)
err = client.Withdraw(ctx, "USDT", &amount)
```

### Init Rules

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

Check `result.Response.OK` for success. On failure, `result.Response.Trace` contains the contract exit code for debugging.

## Key Management

```go
// Register signing key in Smart Account (on-chain transaction)
err := client.AddPublicKey(ctx)
```

Requires `WithSigner()`, `WithWallet()`, `WithSmartAccount()`, and `WithTONApi()`.

## Market Data (On-Chain)

Query vAMM and Vault contracts directly:

```go
// Prices
spotPrice, _ := client.GetSpotPrice(ctx, market)
terminalPrice, _ := client.GetTerminalAmmPrice(ctx, market)

// Market state
state, _ := client.GetAmmState(ctx, market)
status, _ := client.GetAmmStatus(ctx, market)
settings, _ := client.GetExchangeSettings(ctx, market)

// Oracle
oracle, _ := client.GetOracleData(ctx, market)

// Funding
funding, _ := client.GetFunding(ctx, market, oraclePrice, settlementPrice)
premium, _ := client.GetPremium(ctx, market, oraclePrice)

// Vault
vaultData, _ := client.GetVaultData(ctx, "USDT")
bufferData, _ := client.GetBufferData(ctx, "USDT")
```

## Account Data

```go
// Balances (from sequencer)
balances, _ := client.GetBalances(ctx)

// Positions (from sequencer)
positions, _ := client.GetPositions(ctx)
```

### Low-Level On-Chain Queries

```go
// Smart Account
saClient := smartaccount.NewClient(tonAPI, saAddr)
keysData, _ := saClient.GetKeysData(ctx)
balance, _ := saClient.GetBalance(ctx, vaultAddr)
nftData, _ := saClient.GetNftData(ctx)
position, _ := saClient.GetPosition(ctx, vammAddr, 0) // 0=long, 1=short

// Factory
factory := smartaccount.NewFactory(tonAPI, factoryAddr)
factoryData, _ := factory.GetFactoryData(ctx)
minFees, _ := factory.GetMinFees(ctx)

// Position Manager
pm := positionmanager.NewClient(tonAPI, pmAddr)
pmData, _ := pm.GetPositionManagerData(ctx)
inited, _ := pm.GetIsInited(ctx)

// vAMM (direct)
vc := vamm.NewClient(tonAPI, vammAddr)
spotPrice, _ := vc.GetSpotPrice(ctx)
marginData, _ := vc.GetRemainMargin(ctx, oraclePrice, positionCell, settlementPrice)

// Vault (direct)
vault := vault.NewClient(tonAPI, vaultAddr)
vaultData, _ := vault.GetVaultData(ctx)
posAddr, _ := vault.GetPositionAddress(ctx, traderAddr, vammAddr)
```

## Sequencer API

The `sequencer` package wraps the REST API:

```go
import "github.com/storm-trade/sdk-go/sequencer"

seq := sequencer.NewClient(sequencer.TestnetURL)

// Account
state, _ := seq.GetAccountState(ctx, saAddress)
balances, _ := seq.GetBalances(ctx, saAddress)
positions, _ := seq.GetPositions(ctx, saAddress)
nextQID, _ := seq.GetNextQueryID(ctx, saAddress)

// Status
status, _ := seq.GetStatus(ctx)
intent, _ := seq.GetIntent(ctx, hash)

// Orders
resp, _ := seq.PlaceOrder(ctx, placeOrderRequest)
seq.CancelOrder(ctx, cancelOrderRequest)

// Orderbook
depth, _ := seq.GetOrderbookDepth(ctx, assetID, levels)

// Events
events, _ := seq.GetEvents(ctx, sequencer.EventsFilter{})
events, _ := seq.GetAccountEvents(ctx, saAddress, sequencer.EventsFilter{})

// Bundles
bundles, _ := seq.GetBundles(ctx)
finalizing, _ := seq.GetFinalizingBundles(ctx, saAddress)
sent, _ := seq.GetSentBundles(ctx, saAddress)

// Gasless
balance, _ := seq.GetGaslessBalance(ctx, saAddress)
resp, _ = seq.GaslessWithdraw(ctx, withdrawRequest)

// Position sync
seq.SyncPosition(ctx, syncRequest)
```

### Base URLs

| Network | URL |
|---------|-----|
| Testnet | `https://api.stage.stormtrade.dev/instant-trading` |
| Mainnet | `https://api5.storm.tg/instant-trading` |

## CLI Tool

The `cmd/example` directory contains a CLI for testing:

```bash
cp .env.example .env
# Edit .env with your seed and private key

go run ./cmd/example keygen                              # Generate ED25519 key pair
go run ./cmd/example add-key                             # Register key in Smart Account
go run ./cmd/example markets                             # List markets
go run ./cmd/example balance                             # Show balances
go run ./cmd/example positions                           # Show positions
go run ./cmd/example info BTC USDT                       # Market & vault data
go run ./cmd/example deposit USDT 100                    # Deposit
go run ./cmd/example deposit TON 0.5 --init              # First deposit (init SA + key)
go run ./cmd/example withdraw USDT 50                    # Withdraw
go run ./cmd/example order market BTC USDT long 100 3    # Market order
go run ./cmd/example order limit BTC USDT long 100 3 --limit=65000
go run ./cmd/example close BTC USDT long 0.001           # Partial close
go run ./cmd/example close-all BTC USDT long             # Full close
go run ./cmd/example stop-loss BTC USDT long 0.001 60000
go run ./cmd/example take-profit BTC USDT long 0.001 80000
go run ./cmd/example add-margin BTC USDT long 50
go run ./cmd/example remove-margin BTC USDT long 20
go run ./cmd/example cancel <hash>
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `STORM_SEED` | Wallet seed phrase (24 words) | required |
| `STORM_PRIVATE_KEY` | ED25519 private key (hex) | required for orders |
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

| Code | Description |
|------|-------------|
| 115 | Key init not allowed without init flag |
| 170 | Invalid `created_at` timestamp |
| 171 | Query already processed |
| 402 | Public key not registered |
| 411 | Wrong order size |
| 471 | Position not ready for close (cooldown) |

## Links

- [Storm Trade](https://storm.tg)
- [Documentation](https://docs.storm.tg)
- [Telegram](https://t.me/StormTradeBot)

## License

MIT
