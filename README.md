# Storm Trade Go SDK

Go SDK for interacting with [Storm Trade](https://storm.tg) perpetual futures protocol on TON blockchain.

## Installation

```bash
go get github.com/storm-trade/sdk-go
```

## Features

- Smart Account management (deposits, withdrawals, public keys)
- Order creation and signing (market, limit, stop-loss, take-profit)
- Position data parsing
- TLB serialization for TON blockchain

## Contract Addresses

### Factory (Smart Account deployer)

| Network     | Address                                            |
|-------------|----------------------------------------------------|
| **Mainnet** | `EQA34l2ywiFdu_kb-HZMqLngFVDjw0DJZHo1aBokOap8xVMU` |
| **Testnet** | `kQDrG1ZEn3BKkFLAcj1o2bDtlyKDxHCWAyhbTqQxLmk3_Qvr` |

### Markets & Vaults

Market (vAMM) and Vault addresses can be fetched dynamically
using [config-discovery-client](https://github.com/storm-trade/config-discovery-client):

```go
import "github.com/storm-trade/config-discovery-client/client"

// Initialize config client
cfg := client.NewClient(client.Options{
	// Mainnet
	ConfigURL: "https://api5.storm.tg/api/config",
	// Testnet: "https://api.stage.stormtrade.dev/api/config"
})

// Get all markets
markets, err := cfg.GetMarkets()
for _, m := range markets {
	fmt.Printf("Market: %s, vAMM: %s\n", m.Name, m.VammAddress)
}

// Get vault address
vault, err := cfg.GetVault()
fmt.Printf("Vault: %s\n", vault.Address)
```

## Quick Start

### Initialize Client

```go
package main

import (
	"context"
	"fmt"
	"github.com/storm-trade/sdk-go/client/smartaccount"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/ton"
)

func main() {
	// Connect to TON
	client := liteclient.NewConnectionPool()

	// Mainnet
	err := client.AddConnectionsFromConfigUrl(context.Background(),
		"https://ton.org/global.config.json")

	// Or Testnet
	// err := client.AddConnectionsFromConfigUrl(context.Background(),
	//     "https://ton.org/testnet-global.config.json")

	if err != nil {
		panic(err)
	}

	api := ton.NewAPIClient(client, ton.ProofCheckPolicyFast).WithRetry()

	// Initialize Smart Account client
	saAddress := address.MustParseAddr("EQ...your_smart_account_address...")
	saClient := smartaccount.NewClient(api, saAddress)

	// Get account data
	data, err := saClient.GetStorageData(context.Background())
	if err != nil {
		panic(err)
	}

	// Access positions
	for _, pos := range data.Positions.Slice() {
		fmt.Printf("Position: size=%s, direction=%s\n",
			pos.Size.String(),
			pos.GetDirection())
	}
}
```

### Deposit Native TON

```go
import (
	"strings"
	"github.com/storm-trade/sdk-go/client/smartaccount"
	sa "github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

// Create wallet from seed
seed := strings.Split("your seed phrase here", " ")
w, err := wallet.FromSeed(api, seed, wallet.HighloadV2R2)
if err != nil {
	panic(err)
}

// Deposit 10 TON
amount := tlb.MustFromTON("10")

// Optional: add public key for signing orders
pubKey := sa.PublicKey(yourEd25519PublicKey)

tx, err := saClient.DepositNative(
	w,                 // wallet
	w.WalletAddress(), // owner
	vaultAddress,      // vault contract address
	amount,            // amount to deposit
	true,              // init (true for first deposit)
	pubKey,            // optional public keys
)
```

### Deposit Jettons (USDT)

```go
tx, err := saClient.DepositJetton(
	w,                   // wallet
	w.WalletAddress(),   // owner
	vaultAddress,        // vault contract address
	jettonMasterAddress, // USDT master contract
	amount,              // amount to deposit
	true,                // init
	pubKey,              // optional public keys
)
```

### Withdraw

```go
amount := tlb.MustFromTON("5")
tx, err := saClient.Withdraw(w, vaultAddress, amount)
```

### Public Key Management

```go
// Add public key (for signing orders)
tx, err := saClient.AddPublicKey(w, pubKey)

// Remove public key
tx, err := saClient.RemovePublicKey(w, pubKey)

// Remove all keys except current
tx, err := saClient.RemoveAllExceptCurrentPublicKey(w, currentPubKey)
```

## Creating Orders

### Order Types

```go
import (
	"time"
	"github.com/storm-trade/sdk-go/tlb"
)

// Market Order - execute immediately at market price
marketOrder := tlb.Order{
	Value: tlb.MarketOrder{
		Payload: tlb.LimitOrderData{
			Expiration:       uint32(time.Now().Add(5 * time.Minute).Unix()),
			Direction:        uint(tlb.ContractDirectionLong), // 0 = long, 1 = short
			Amount:           tlb.MustFromTON("100"),          // position size in USD
			Leverage:         3_000_000_000,                   // 3x leverage (9 decimals)
			LimitPrice:       tlb.MustFromTON("0"),            // 0 for market
			StopPrice:        tlb.MustFromTON("0"),
			StopTriggerPrice: tlb.MustFromTON("0"),
			TakeTriggerPrice: tlb.MustFromTON("0"),
		},
	},
}

// Limit Order - execute when price reaches limit
limitOrder := tlb.Order{
	Value: tlb.LimitOrder{
		Payload: tlb.LimitOrderData{
			Expiration:       uint32(time.Now().Add(24 * time.Hour).Unix()),
			Direction:        uint(tlb.ContractDirectionLong),
			Amount:           tlb.MustFromTON("100"),
			Leverage:         5_000_000_000,           // 5x
			LimitPrice:       tlb.MustFromTON("95000"), // entry price
			StopPrice:        tlb.MustFromTON("0"),
			StopTriggerPrice: tlb.MustFromTON("0"),
			TakeTriggerPrice: tlb.MustFromTON("0"),
		},
	},
}

// Stop-Loss Order
stopLossOrder := tlb.Order{
	Value: tlb.StopOrder{
		Payload: tlb.StopOrderData{
			Expiration:   uint32(time.Now().Add(7 * 24 * time.Hour).Unix()),
			Direction:    uint(tlb.ContractDirectionLong),
			Amount:       tlb.MustFromTON("100"),   // close amount
			TriggerPrice: tlb.MustFromTON("90000"), // trigger price
		},
	},
}

// Take-Profit Order
takeProfitOrder := tlb.Order{
	Value: tlb.TakeOrder{
		Payload: tlb.StopOrderData{
			Expiration:   uint32(time.Now().Add(7 * 24 * time.Hour).Unix()),
			Direction:    uint(tlb.ContractDirectionLong),
			Amount:       tlb.MustFromTON("100"),
			TriggerPrice: tlb.MustFromTON("110000"),
		},
	},
}

// Add Margin
addMarginOrder := tlb.Order{
	Value: tlb.AddMarginOrder{
		Payload: tlb.MarginOrderData{
			Direction: uint(tlb.ContractDirectionLong),
			Amount:    tlb.MustFromTON("50"), // margin to add
		},
	},
}

// Remove Margin
removeMarginOrder := tlb.Order{
	Value: tlb.RemoveMarginOrder{
		Payload: tlb.MarginOrderData{
			Direction: uint(tlb.ContractDirectionLong),
			Amount:    tlb.MustFromTON("25"), // margin to remove
		},
	},
}
```

### Create and Sign User Intent

```go
import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"time"
	"github.com/storm-trade/sdk-go/contracts/hw"
	"github.com/storm-trade/sdk-go/contracts/smartaccount"
	gotlb "github.com/xssnick/tonutils-go/tlb"
)

// Create UserIntent
intent := &smartaccount.UserIntent{
	QueryId:          hw.FromSeqno(nextQueryId), // get from sequencer API
	CreatedAt:        uint64(time.Now().Unix()),
	ReferenceQueryId: nil,            // for linked orders
	PublicKey:        publicKeyBytes, // 32 bytes
	Intent: &smartaccount.UserIntentPayload{
		VAmm:         vammAddress, // market contract
		SmartAccount: saAddress,
		Order:        &marketOrder,
	},
}

// Sign the intent
privateKey := ed25519.PrivateKey(yourPrivateKeyBytes)
signedMessage, err := smartaccount.SignMessage(intent, privateKey)
if err != nil {
	panic(err)
}

// Get intent hash
hash, err := signedMessage.Hash()
fmt.Println("Intent hash:", hash)

// Serialize to cell for sending
cell, err := gotlb.ToCell(signedMessage)
if err != nil {
	panic(err)
}

// Base64 encode for API
msgBase64 := base64.StdEncoding.EncodeToString(cell.ToBOC())
```

### Cancel Order

```go
// Create cancel message
cancelMsg := &smartaccount.CancelMessage{
	SmartAccountAddress: saAddress,
	OrderId:             orderHashBytes, // 32 bytes - hash of the order to cancel
}

// Sign it
cancelCell, _ := gotlb.ToCell(cancelMsg)
signature := ed25519.Sign(privateKey, cancelCell.Hash())

signedCancel := &smartaccount.SignedCancelMessage{
	Message:   cancelMsg,
	PublicKey: publicKeyBytes,
	Signature: signature,
}
```

## Send Orders to Sequencer

Orders are sent to the Storm Trade sequencer via REST API:

```go
import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
)

type PlaceOrderRequest struct {
	SmartAccount string `json:"sa"`
	Message      string `json:"message"`    // base64 encoded signed intent
	PublicKey    string `json:"public_key"` // base64 encoded public key
	Signature    string `json:"signature"`  // base64 encoded signature
}

func placeOrder(signedMessage *smartaccount.SignedMessage) error {
	cell, _ := gotlb.ToCell(signedMessage.Message)

	req := PlaceOrderRequest{
		SmartAccount: saAddress.String(),
		Message:      base64.StdEncoding.EncodeToString(cell.ToBOC()),
		PublicKey:    base64.StdEncoding.EncodeToString(signedMessage.PublicKey[:]),
		Signature:    base64.StdEncoding.EncodeToString(signedMessage.Signature),
	}

	body, _ := json.Marshal(req)

	// Mainnet: https://api5.storm.tg/instant-trading
	// Testnet: https://api.stage.stormtrade.dev/instant-trading
	resp, err := http.Post(
		"https://api5.storm.tg/instant-trading/order/place",
		"application/json",
		bytes.NewReader(body),
	)

	return err
}
```

## Data Structures

### Position State

```go
type PositionState struct {
	Size                         *big.Int   // position size (9 decimals)
	Direction                    uint8      // 0 = long, 1 = short
	Margin                       *tlb.Coins // margin amount
	OpenNotional                 *tlb.Coins // open notional value
	LastUpdatedCumulativePremium int64      // funding rate accumulator
	Fee                          uint64     // trading fee (basis points)
	Discount                     uint64     // fee discount
	Rebate                       uint64     // referral rebate
	LastUpdatedTimestamp         uint64     // last update time
}

// Check position direction
pos.GetDirection() // returns tlb.DirectionLong or tlb.DirectionShort

// Check if position is closed
pos.IsClosed() // returns true if Size == 0
```

### Query ID

Storm Trade uses a custom QueryId format for replay protection:

```go
import "github.com/storm-trade/sdk-go/contracts/hw"

// Create from sequence number (get next from API)
queryId := hw.FromSeqno(12345)

// Get sequence number from QueryId
seqno := queryId.Seqno()

// Max query ID
maxId := hw.MaxQueryId // 1023 * 8192 = 8,380,416
```

## API Endpoints

Base URLs:

- **Mainnet:** `https://api5.storm.tg/instant-trading`
- **Testnet:** `https://api.stage.stormtrade.dev/instant-trading`

| Endpoint                           | Method | Description          |
|------------------------------------|--------|----------------------|
| `/order/place`                     | POST   | Place a new order    |
| `/order/cancel`                    | POST   | Cancel an order      |
| `/status`                          | GET    | Get sequencer status |
| `/smartaccount/{address}/state`    | GET    | Get account state    |
| `/smartaccount/{address}/balance`  | GET    | Get account balance  |
| `/smartaccount/{address}/query_id` | GET    | Get next query ID    |
| `/orderbook/orders`                | GET    | Get orderbook orders |
| `/intent/{hash}`                   | GET    | Get intent by hash   |

## Constants

### Leverage

Leverage is specified with 9 decimal places:

- 1x = `1_000_000_000`
- 2x = `2_000_000_000`
- 5x = `5_000_000_000`
- 10x = `10_000_000_000`
- Max = `50_000_000_000` (50x)

### Direction

```go
tlb.ContractDirectionLong  = 0 // Long position
tlb.ContractDirectionShort = 1 // Short position
```

### Order Types

```go
tlb.MarketOrderType       // Execute at market price
tlb.LimitOrderType        // Execute at limit price
tlb.StopOrderType         // Stop-loss order
tlb.TakeOrderType         // Take-profit order
tlb.AddMarginOrderType    // Add margin to position
tlb.RemoveMarginOrderType // Remove margin from position
```

## Error Handling

Common smart account errors:

| Code | Name                   | Description               |
|------|------------------------|---------------------------|
| 171  | QueryAlreadyProcessed  | Query ID already used     |
| 174  | IntentAlreadyProcessed | Intent already executed   |
| 402  | PublicKeyNotFound      | Public key not registered |
| 409  | InvalidPosition        | Position state mismatch   |
| 411  | WrongSize              | Invalid order size        |
| 431  | InvalidBaseAssetAmount | Invalid amount            |

## Examples

See the [examples](./examples) directory for complete working examples.

## Links

- [Storm Trade](https://storm.tg)
- [Documentation](https://docs.storm.tg)
- [API Reference](https://api5.storm.tg/instant-trading/swagger/index.html)
- [Telegram](https://t.me/stormtrade)

## License

MIT
