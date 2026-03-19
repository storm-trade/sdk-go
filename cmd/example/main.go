package main

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/storm-trade/sdk-go/client/smartaccount"
	"github.com/storm-trade/sdk-go/storm"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

func main() {
	loadEnvFile(".env")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "keygen":
		cmdKeygen()
	case "markets":
		cmdMarkets(ctx)
	case "balance":
		cmdBalance(ctx)
	case "positions":
		cmdPositions(ctx)
	case "add-key":
		cmdAddKey(ctx)
	case "deposit":
		cmdDeposit(ctx)
	case "order":
		cmdOrder(ctx)
	case "cancel":
		cmdCancel(ctx)
	case "withdraw":
		cmdWithdraw(ctx)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`Storm Trade SDK — Example CLI

Usage: go run ./cmd/example <command>

Commands:
  keygen      Generate ED25519 key pair for order signing
  add-key     Add public key to smart account (on-chain)
  markets     List available markets
  balance     Show smart account balances
  positions   Show open positions
  deposit     Deposit to smart account (e.g. deposit TON 1, deposit USDT 10)
  withdraw    Withdraw from smart account (e.g. withdraw TON 1)
  order       Place an order (e.g. order market BTC USDT long 1000 3)
  cancel      Cancel order by hash

Environment:
  STORM_SEED          Wallet seed phrase
  STORM_PRIVATE_KEY   Hex-encoded ED25519 private key`)
}

func mustClient(ctx context.Context) *storm.Client {
	network := storm.Testnet
	var opts []storm.Option

	client := storm.NewClient(network)

	if seed := os.Getenv("STORM_SEED"); seed != "" {
		api, w := mustTON(ctx, seed)
		opts = append(opts, storm.WithTONApi(api), storm.WithWallet(w))

		factory := smartaccount.NewFactory(api, address.MustParseAddr(client.FactoryAddress()))
		saAddr, err := factory.GetSmartAccountAddress(ctx, w.WalletAddress())
		if err != nil {
			log.Fatalf("get smart account address: %v", err)
		}
		opts = append(opts, storm.WithSmartAccount(saAddr))
		fmt.Printf("Smart Account: %s\n", saAddr.String())
	}

	if keyHex := os.Getenv("STORM_PRIVATE_KEY"); keyHex != "" {
		keyBytes, err := hex.DecodeString(keyHex)
		if err != nil {
			log.Fatalf("parse private key: %v", err)
		}
		opts = append(opts, storm.WithSigner(ed25519.PrivateKey(keyBytes)))
	}

	opts = append(opts, storm.WithClockSkew(5*time.Second))

	return storm.NewClient(network, opts...)
}

func mustTON(ctx context.Context, seed string) (ton.APIClientWrapped, *wallet.Wallet) {
	pool := liteclient.NewConnectionPool()
	if err := pool.AddConnectionsFromConfigUrl(ctx, "https://ton-blockchain.github.io/testnet-global.config.json"); err != nil {
		log.Fatalf("connect to TON: %v", err)
	}

	api := ton.NewAPIClient(pool, ton.ProofCheckPolicyUnsafe).WithRetry(10)
	words := strings.Split(seed, " ")
	w, err := wallet.FromSeed(api, words, wallet.V4R2)
	if err != nil {
		log.Fatalf("wallet from seed: %v", err)
	}

	return api, w
}

func cmdKeygen() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}

	privHex := hex.EncodeToString(priv)
	pubHex := hex.EncodeToString(pub)

	fmt.Printf("Private key: %s\n", privHex)
	fmt.Printf("Public key:  %s\n", pubHex)
	fmt.Println()
	fmt.Println("Save the private key and set it as environment variable:")
	fmt.Printf("  export STORM_PRIVATE_KEY=%s\n", privHex)
}

func cmdAddKey(ctx context.Context) {
	client := mustClient(ctx)
	if err := client.AddPublicKey(ctx); err != nil {
		log.Fatalf("add key: %v", err)
	}
	fmt.Println("Public key added to smart account")
}

func cmdMarkets(ctx context.Context) {
	client := storm.NewClient(storm.Testnet)
	markets, err := client.Markets()
	if err != nil {
		log.Fatalf("fetch markets: %v", err)
	}
	for _, m := range markets {
		hidden := ""
		if m.IsHidden {
			hidden = " [hidden]"
		}
		fmt.Printf("  %-20s %-20s type=%-10s settlement=%s%s\n", m.Name, m.Ticker, m.Type, m.SettlementToken, hidden)
	}
}

func cmdBalance(ctx context.Context) {
	client := mustClient(ctx)
	balances, err := client.GetBalances(ctx)
	if err != nil {
		log.Fatalf("get balances: %v", err)
	}
	for addr, amount := range balances {
		fmt.Printf("  %s: %d\n", addr, amount)
	}
}

func cmdPositions(ctx context.Context) {
	client := mustClient(ctx)
	positions, err := client.GetPositions(ctx)
	if err != nil {
		log.Fatalf("get positions: %v", err)
	}
	if len(positions) == 0 {
		fmt.Println("  No open positions")
		return
	}
	for _, p := range positions {
		fmt.Printf("  %s %s (status: %s)\n", p.Market, p.Direction, p.State.Status)
	}
}

func cmdDeposit(ctx context.Context) {
	if len(os.Args) < 4 {
		log.Fatal("usage: deposit <asset> <amount>")
	}

	asset := os.Args[2]
	client := mustClient(ctx)

	a, err := client.Asset(asset)
	if err != nil {
		log.Fatalf("get asset: %v", err)
	}

	amount := tlb.MustFromDecimal(os.Args[3], a.Decimals)
	if err := client.Deposit(ctx, asset, &amount); err != nil {
		log.Fatalf("deposit: %v", err)
	}
	fmt.Printf("Deposited %s %s\n", os.Args[3], asset)
}

func cmdWithdraw(ctx context.Context) {
	if len(os.Args) < 4 {
		log.Fatal("usage: withdraw <asset> <amount>")
	}

	asset := os.Args[2]
	client := mustClient(ctx)

	a, err := client.Asset(asset)
	if err != nil {
		log.Fatalf("get asset: %v", err)
	}

	amount := tlb.MustFromDecimal(os.Args[3], a.Decimals)
	if err := client.Withdraw(ctx, asset, &amount); err != nil {
		log.Fatalf("withdraw: %v", err)
	}
	fmt.Printf("Withdrawn %s %s\n", os.Args[3], asset)
}

func cmdOrder(ctx context.Context) {
	if len(os.Args) < 8 {
		log.Fatal("usage: order <market|limit|stop-limit> <asset> <settlement> <long|short> <amount> <leverage> [--limit=X] [--stop=X] [--sl=X] [--tp=X]")
	}

	orderType := os.Args[2]
	assetName := os.Args[3]
	settlement := os.Args[4]
	dirStr := os.Args[5]
	amountStr := os.Args[6]
	leverageStr := os.Args[7]

	var dir storm.Direction
	switch dirStr {
	case "long":
		dir = storm.Long
	case "short":
		dir = storm.Short
	default:
		log.Fatalf("invalid direction %q, use long or short", dirStr)
	}

	leverage, err := strconv.ParseUint(leverageStr, 10, 64)
	if err != nil {
		log.Fatalf("invalid leverage: %v", err)
	}
	leverage *= 1_000_000_000

	client := mustClient(ctx)
	market, err := client.Market(assetName, settlement)
	if err != nil {
		log.Fatalf("get market: %v", err)
	}

	asset, err := client.Asset(settlement)
	if err != nil {
		log.Fatalf("get asset: %v", err)
	}

	amount := tlb.MustFromDecimal(amountStr, asset.Decimals)

	var opts []storm.Option
	var limitPrice, stopPrice string
	for _, arg := range os.Args[8:] {
		if v, ok := strings.CutPrefix(arg, "--sl="); ok {
			price := tlb.MustFromTON(v)
			opts = append(opts, storm.WithStopLoss(&price))
		}
		if v, ok := strings.CutPrefix(arg, "--tp="); ok {
			price := tlb.MustFromTON(v)
			opts = append(opts, storm.WithTakeProfit(&price))
		}
		if v, ok := strings.CutPrefix(arg, "--limit="); ok {
			limitPrice = v
		}
		if v, ok := strings.CutPrefix(arg, "--stop="); ok {
			stopPrice = v
		}
	}

	var resp *storm.PlaceOrderResult
	switch orderType {
	case "market":
		resp, err = client.PlaceMarketOrder(ctx, market, dir, &amount, leverage, opts...)
	case "limit":
		if limitPrice == "" {
			log.Fatal("--limit=<price> required for limit order")
		}
		lp := tlb.MustFromTON(limitPrice)
		resp, err = client.PlaceLimitOrder(ctx, market, dir, &amount, leverage, &lp, opts...)
	case "stop-limit":
		if limitPrice == "" || stopPrice == "" {
			log.Fatal("--limit=<price> and --stop=<price> required for stop-limit order")
		}
		lp := tlb.MustFromTON(limitPrice)
		sp := tlb.MustFromTON(stopPrice)
		resp, err = client.PlaceStopLimitOrder(ctx, market, dir, &amount, leverage, &lp, &sp, opts...)
	default:
		log.Fatalf("unknown order type %q, use market, limit, or stop-limit", orderType)
	}

	if err != nil {
		log.Fatalf("place order: %v", err)
	}
	fmt.Printf("Order placed: ok=%v\n", resp.Response.OK)
	fmt.Printf("  hash=%x\n", resp.OrderHash)
	if resp.StopLossHash != nil {
		fmt.Printf("  stopLoss hash=%x\n", resp.StopLossHash)
	}
	if resp.TakeProfitHash != nil {
		fmt.Printf("  takeProfit hash=%x\n", resp.TakeProfitHash)
	}
}

func cmdCancel(ctx context.Context) {
	if len(os.Args) < 3 {
		log.Fatal("usage: cancel <orderHash>")
	}

	hashBytes, err := hex.DecodeString(os.Args[2])
	if err != nil {
		log.Fatalf("invalid order hash: %v", err)
	}

	client := mustClient(ctx)
	if err := client.CancelOrder(ctx, hashBytes); err != nil {
		log.Fatalf("cancel order: %v", err)
	}
	fmt.Printf("Order %s cancelled\n", os.Args[2])
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)

		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}
