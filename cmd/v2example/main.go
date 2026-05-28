package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	pmclient "github.com/storm-trade/sdk-go/client/positionmanager"
	vaultclient "github.com/storm-trade/sdk-go/client/vault"
	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/stormv2"
	stlb "github.com/storm-trade/sdk-go/tlb"
	"github.com/xssnick/tonutils-go/liteclient"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
	"github.com/xssnick/tonutils-go/ton/wallet"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

func main() {
	loadEnvFile(".env")

	if len(os.Args) < 2 {
		printUsage()
		return
	}

	ctx := context.Background()

	switch os.Args[1] {
	case "markets":
		cmdMarkets(ctx)
	case "order":
		cmdOrder(ctx)
	case "close":
		cmdClose(ctx)
	case "stop-loss":
		cmdStopLoss(ctx)
	case "take-profit":
		cmdTakeProfit(ctx)
	case "add-margin":
		cmdAddMargin(ctx)
	case "remove-margin":
		cmdRemoveMargin(ctx)
	case "cancel":
		cmdCancel(ctx)
	case "orders":
		cmdOrders(ctx)
	default:
		printUsage()
	}
}

func printUsage() {
	fmt.Println(`Storm Trade v2 SDK — Example CLI

Usage: go run ./cmd/v2example <command>

Commands:
  markets       List available markets
  order         Place an order (order market BTC TON long 100 3)
  close         Close position (close BTC TON long 0.001)
  stop-loss     Set stop loss (stop-loss BTC TON long 0.001 60000)
  take-profit   Set take profit (take-profit BTC TON long 0.001 80000)
  add-margin    Add margin (add-margin BTC TON long 50)
  remove-margin Remove margin (remove-margin BTC TON long 20)
  cancel        Cancel order (cancel BTC TON long <orderType> <orderIndex>)
  orders        Show active orders (orders BTC USDT long)

Environment:
  STORM_SEED            Wallet seed phrase
  STORM_NETWORK         testnet (default) or mainnet
  STORM_WALLET_VERSION  v3r2, v4r2 (default), v5r1
  STORM_SUBWALLET_ID    Custom subwallet ID (optional)
  STORM_GLOBAL_ID       Override network global ID for v5r1 (optional)`)
}

func parseNetwork() config.Network {
	switch os.Getenv("STORM_NETWORK") {
	case "", "testnet":
		return config.Testnet
	case "mainnet":
		return config.Mainnet
	default:
		log.Fatalf("unknown STORM_NETWORK %q", os.Getenv("STORM_NETWORK"))
		return 0
	}
}

func parseDirection(s string) config.Direction {
	switch s {
	case "long":
		return config.Long
	case "short":
		return config.Short
	default:
		log.Fatalf("invalid direction %q, use long or short", s)
		return 0
	}
}

var tonConfigURLs = map[config.Network]string{
	config.Testnet: "https://ton-blockchain.github.io/testnet-global.config.json",
	config.Mainnet: "https://ton-blockchain.github.io/global.config.json",
}

var networkGlobalIDs = map[config.Network]int32{
	config.Testnet: wallet.TestnetGlobalID,
	config.Mainnet: wallet.MainnetGlobalID,
}

func mustClient(ctx context.Context) *stormv2.Client {
	network := parseNetwork()
	var opts []stormv2.Option

	if seed := os.Getenv("STORM_SEED"); seed != "" {
		api, w := mustTON(ctx, seed, network)
		opts = append(opts, stormv2.WithTONApi(api), stormv2.WithWallet(w))

		walletVer := os.Getenv("STORM_WALLET_VERSION")
		if walletVer == "" {
			walletVer = "v4r2"
		}
		fmt.Printf("Network: %s | Wallet: %s (subwallet=%d) | Address: %s\n",
			os.Getenv("STORM_NETWORK"), walletVer, w.GetSubwalletID(), w.WalletAddress())
	}

	return stormv2.NewClient(network, opts...)
}

func mustTON(ctx context.Context, seed string, network config.Network) (ton.APIClientWrapped, *wallet.Wallet) {
	pool := liteclient.NewConnectionPool()
	if err := pool.AddConnectionsFromConfigUrl(ctx, tonConfigURLs[network]); err != nil {
		log.Fatalf("connect to TON: %v", err)
	}

	api := ton.NewAPIClient(pool, ton.ProofCheckPolicyUnsafe).WithRetry(10)
	words := strings.Split(seed, " ")

	ver := walletVersion(network)
	w, err := wallet.FromSeed(api, words, ver)
	if err != nil {
		log.Fatalf("wallet from seed: %v", err)
	}

	if idStr := os.Getenv("STORM_SUBWALLET_ID"); idStr != "" {
		id, err := strconv.ParseUint(idStr, 10, 32)
		if err != nil {
			log.Fatalf("invalid STORM_SUBWALLET_ID: %v", err)
		}
		w, err = w.GetSubwallet(uint32(id))
		if err != nil {
			log.Fatalf("get subwallet: %v", err)
		}
	}

	return api, w
}

func walletVersion(network config.Network) wallet.VersionConfig {
	switch os.Getenv("STORM_WALLET_VERSION") {
	case "", "v4r2":
		return wallet.V4R2
	case "v3r2":
		return wallet.V3R2
	case "v4r1":
		return wallet.V4R1
	case "v5r1":
		globalID := networkGlobalIDs[network]
		if idStr := os.Getenv("STORM_GLOBAL_ID"); idStr != "" {
			id, err := strconv.ParseInt(idStr, 10, 32)
			if err != nil {
				log.Fatalf("invalid STORM_GLOBAL_ID: %v", err)
			}
			globalID = int32(id)
		}
		return wallet.ConfigV5R1Final{
			NetworkGlobalID: globalID,
			Workchain:       0,
		}
	default:
		log.Fatalf("unknown STORM_WALLET_VERSION %q", os.Getenv("STORM_WALLET_VERSION"))
		return nil
	}
}

func printResult(result *stormv2.TxResult) {
	fmt.Printf("TX hash: %s\n", result.Hash)
}

func cmdMarkets(ctx context.Context) {
	client := stormv2.NewClient(parseNetwork())
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

func cmdOrder(ctx context.Context) {
	if len(os.Args) < 8 {
		log.Fatal("usage: order <market|limit|stop-limit> <asset> <settlement> <long|short> <amount> <leverage> [--limit=X] [--stop=X]")
	}

	orderType := os.Args[2]
	assetName := os.Args[3]
	settlement := os.Args[4]
	dir := parseDirection(os.Args[5])
	amountStr := os.Args[6]
	leverageStr := os.Args[7]

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

	var limitPrice, stopPrice string
	var opts []stormv2.Option
	for _, arg := range os.Args[8:] {
		if v, ok := strings.CutPrefix(arg, "--limit="); ok {
			limitPrice = v
		}
		if v, ok := strings.CutPrefix(arg, "--stop="); ok {
			stopPrice = v
		}
		if v, ok := strings.CutPrefix(arg, "--sl="); ok {
			price := tlb.MustFromTON(v)
			opts = append(opts, stormv2.WithStopLoss(&price))
		}
		if v, ok := strings.CutPrefix(arg, "--tp="); ok {
			price := tlb.MustFromTON(v)
			opts = append(opts, stormv2.WithTakeProfit(&price))
		}
		if v, ok := strings.CutPrefix(arg, "--ref="); ok {
			id, err := strconv.ParseUint(v, 10, 64)
			if err != nil {
				log.Fatalf("invalid --ref: %v", err)
			}
			opts = append(opts, stormv2.WithReferralID(id))
		}
	}

	var result *stormv2.TxResult
	switch orderType {
	case "market":
		result, err = client.PlaceMarketOrder(ctx, market, dir, &amount, leverage, opts...)
	case "limit":
		if limitPrice == "" {
			log.Fatal("--limit=<price> required")
		}
		lp := tlb.MustFromTON(limitPrice)
		result, err = client.PlaceLimitOrder(ctx, market, dir, &amount, leverage, &lp, opts...)
	case "stop-limit":
		if limitPrice == "" || stopPrice == "" {
			log.Fatal("--limit=<price> and --stop=<price> required")
		}
		lp := tlb.MustFromTON(limitPrice)
		sp := tlb.MustFromTON(stopPrice)
		result, err = client.PlaceStopLimitOrder(ctx, market, dir, &amount, leverage, &lp, &sp, opts...)
	default:
		log.Fatalf("unknown order type %q", orderType)
	}
	if err != nil {
		log.Fatalf("place order: %v", err)
	}
	printResult(result)
}

func cmdClose(ctx context.Context) {
	if len(os.Args) < 6 {
		log.Fatal("usage: close <asset> <settlement> <long|short> <size>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	size := tlb.MustFromDecimal(os.Args[5], 9)
	result, err := client.ClosePosition(ctx, market, dir, &size)
	if err != nil {
		log.Fatalf("close: %v", err)
	}
	printResult(result)
}

func cmdStopLoss(ctx context.Context) {
	if len(os.Args) < 7 {
		log.Fatal("usage: stop-loss <asset> <settlement> <long|short> <size> <triggerPrice>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	size := tlb.MustFromDecimal(os.Args[5], 9)
	trigger := tlb.MustFromTON(os.Args[6])
	result, err := client.PlaceStopLoss(ctx, market, dir, &size, &trigger)
	if err != nil {
		log.Fatalf("stop loss: %v", err)
	}
	printResult(result)
}

func cmdTakeProfit(ctx context.Context) {
	if len(os.Args) < 7 {
		log.Fatal("usage: take-profit <asset> <settlement> <long|short> <size> <triggerPrice>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	size := tlb.MustFromDecimal(os.Args[5], 9)
	trigger := tlb.MustFromTON(os.Args[6])
	result, err := client.PlaceTakeProfit(ctx, market, dir, &size, &trigger)
	if err != nil {
		log.Fatalf("take profit: %v", err)
	}
	printResult(result)
}

func cmdAddMargin(ctx context.Context) {
	if len(os.Args) < 6 {
		log.Fatal("usage: add-margin <asset> <settlement> <long|short> <amount>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	asset, err := client.Asset(os.Args[3])
	if err != nil {
		log.Fatalf("get asset: %v", err)
	}
	dir := parseDirection(os.Args[4])
	amount := tlb.MustFromDecimal(os.Args[5], asset.Decimals)
	result, err := client.AddMargin(ctx, market, dir, &amount)
	if err != nil {
		log.Fatalf("add margin: %v", err)
	}
	printResult(result)
}

func cmdRemoveMargin(ctx context.Context) {
	if len(os.Args) < 6 {
		log.Fatal("usage: remove-margin <asset> <settlement> <long|short> <amount>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	amount := tlb.MustFromDecimal(os.Args[5], 9)
	result, err := client.RemoveMargin(ctx, market, dir, &amount)
	if err != nil {
		log.Fatalf("remove margin: %v", err)
	}
	printResult(result)
}

func cmdCancel(ctx context.Context) {
	if len(os.Args) < 7 {
		log.Fatal("usage: cancel <asset> <settlement> <long|short> <orderType> <orderIndex>")
	}
	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	orderType, err := strconv.ParseUint(os.Args[5], 10, 64)
	if err != nil {
		log.Fatalf("invalid orderType: %v", err)
	}
	orderIndex, err := strconv.ParseUint(os.Args[6], 10, 64)
	if err != nil {
		log.Fatalf("invalid orderIndex: %v", err)
	}
	result, err := client.CancelOrder(ctx, market, dir, orderType, orderIndex)
	if err != nil {
		log.Fatalf("cancel: %v", err)
	}
	printResult(result)
}

var orderTypeNames = map[int]string{
	0: "stopLoss", 1: "takeProfit", 2: "limit", 3: "market",
}

func cmdOrders(ctx context.Context) {
	if len(os.Args) < 5 {
		log.Fatal("usage: orders <asset> <settlement> <long|short>")
	}

	client := mustClient(ctx)
	o := client.TonAPI()
	if o == nil {
		log.Fatal("TON API required")
	}

	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])

	w := client.Wallet()
	if w == nil {
		log.Fatal("wallet required")
	}

	fmt.Printf("Wallet:  %s\n", w.WalletAddress())
	fmt.Printf("Vault:   %s\n", market.VaultAddress)
	fmt.Printf("AssetIndex: %d\n", market.AssetIndex)

	vc := vaultclient.NewClient(o, market.VaultAddress)
	vammAddr, err := vc.GetVAMMAddress(ctx, market.AssetIndex)
	if err != nil {
		log.Fatalf("get vamm address: %v", err)
	}
	pmAddr, err := vc.GetPositionAddress(ctx, w.WalletAddress(), vammAddr)
	if err != nil {
		log.Fatalf("get PM address: %v", err)
	}

	fmt.Printf("PM:      %s\n", pmAddr)

	pm := pmclient.NewClient(o, pmAddr)
	data, err := pm.GetPositionManagerData(ctx)
	if err != nil {
		fmt.Printf("Not initialized or error: %v\n", err)
		return
	}
	fmt.Println()

	if data.Orders != nil && data.Orders.BitsSize()+data.Orders.RefsNum() > 0 {
		wrapped := cell.BeginCell().MustStoreMaybeRef(data.Orders).EndCell()
		dict, _ := wrapped.MustBeginParse().LoadDict(3)
		if dict == nil {
			fmt.Println("Limit orders: none")
		} else {
			orders, err := stlb.MapOrders(dict)
			if err != nil {
				fmt.Printf("Limit orders: parse error: %v\n", err)
			} else if len(orders) == 0 {
				fmt.Println("Limit orders: none")
			} else {
				fmt.Println("Limit orders:")
				for idx, ord := range orders {
					typeName := orderTypeNames[int(ord.GetType())]
					fmt.Printf("  index=%d type=%s direction=%s\n", idx, typeName, ord.GetDirection())
				}
			}
		}
	} else {
		fmt.Println("Limit orders: none")
	}

	posCell := data.LongPosition
	label := "long"
	if dir == config.Short {
		posCell = data.ShortPosition
		label = "short"
	}

	if posCell != nil {
		var posRef stlb.PositionRef
		if err := tlb.LoadFromCell(&posRef, posCell.MustBeginParse()); err != nil {
			fmt.Printf("%s position: parse error: %v\n", label, err)
		} else {
			fmt.Printf("\n%s position: size=%s margin=%s\n", label, posRef.Position.Size, posRef.Position.Margin)
			sltpOrders := posRef.Orders()
			if len(sltpOrders) == 0 {
				fmt.Printf("  SL/TP orders: none\n")
			} else {
				for idx, ord := range sltpOrders {
					typeName := orderTypeNames[int(ord.GetType())]
					fmt.Printf("  SL/TP index=%d type=%s\n", idx, typeName)
				}
			}
		}
	} else {
		fmt.Printf("\n%s position: none\n", label)
	}
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
