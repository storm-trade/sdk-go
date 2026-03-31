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

	"math/big"

	pmclient "github.com/storm-trade/sdk-go/client/positionmanager"
	"github.com/storm-trade/sdk-go/client/smartaccount"
	vammclient "github.com/storm-trade/sdk-go/client/vamm"
	vaultclient "github.com/storm-trade/sdk-go/client/vault"
	"github.com/storm-trade/sdk-go/config"
	"github.com/storm-trade/sdk-go/storm"
	stlb "github.com/storm-trade/sdk-go/tlb"
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
	case "close":
		cmdClose(ctx)
	case "close-all":
		cmdCloseAll(ctx)
	case "stop-loss":
		cmdStopLoss(ctx)
	case "take-profit":
		cmdTakeProfit(ctx)
	case "add-margin":
		cmdAddMargin(ctx)
	case "remove-margin":
		cmdRemoveMargin(ctx)
	case "info":
		cmdInfo(ctx)
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
  order       Place an order (order market BTC USDT long 1000 3)
  close       Close position by size (close BTC USDT long 0.001)
  close-all   Close entire position (close-all BTC USDT long)
  stop-loss   Set stop loss (stop-loss BTC USDT long 0.001 60000)
  take-profit Set take profit (take-profit BTC USDT long 0.001 80000)
  add-margin  Add margin (add-margin BTC USDT long 500)
  remove-margin Remove margin (remove-margin BTC USDT long 500)
  cancel      Cancel order by hash
  info        Show market & vault info (info BTC USDT)

Environment:
  STORM_SEED            Wallet seed phrase
  STORM_PRIVATE_KEY     Hex-encoded ED25519 private key
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
		log.Fatalf("unknown STORM_NETWORK %q, use testnet or mainnet", os.Getenv("STORM_NETWORK"))
		return 0
	}
}

func mustClient(ctx context.Context) *storm.Client {
	network := parseNetwork()
	factoryAddr := config.Networks[network].FactoryAddress
	var opts []storm.Option

	if seed := os.Getenv("STORM_SEED"); seed != "" {
		api, w := mustTON(ctx, seed, network)
		opts = append(opts, storm.WithTONApi(api), storm.WithWallet(w))

		factory := smartaccount.NewFactory(api, address.MustParseAddr(factoryAddr))
		saAddr, err := factory.GetSmartAccountAddress(ctx, w.WalletAddress())
		if err != nil {
			log.Fatalf("get smart account address: %v", err)
		}
		opts = append(opts, storm.WithSmartAccount(saAddr))

		walletVer := os.Getenv("STORM_WALLET_VERSION")
		if walletVer == "" {
			walletVer = "v4r2"
		}
		fmt.Printf("Network: %s | Wallet: %s (subwallet=%d) | Address: %s\n", os.Getenv("STORM_NETWORK"), walletVer, w.GetSubwalletID(), w.WalletAddress())
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

var tonConfigURLs = map[config.Network]string{
	config.Testnet: "https://ton-blockchain.github.io/testnet-global.config.json",
	config.Mainnet: "https://ton-blockchain.github.io/global.config.json",
}

var networkGlobalIDs = map[config.Network]int32{
	config.Testnet: wallet.TestnetGlobalID,
	config.Mainnet: wallet.MainnetGlobalID,
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
		log.Fatalf("unknown STORM_WALLET_VERSION %q, use v3r2, v4r2, or v5r1", os.Getenv("STORM_WALLET_VERSION"))
		return nil
	}
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
	client := storm.NewClient(parseNetwork())
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
		log.Fatal("usage: deposit <asset> <amount> [--init]")
	}

	asset := os.Args[2]
	client := mustClient(ctx)

	a, err := client.Asset(asset)
	if err != nil {
		log.Fatalf("get asset: %v", err)
	}

	amount := tlb.MustFromDecimal(os.Args[3], a.Decimals)

	var opts []storm.Option
	for _, arg := range os.Args[4:] {
		if arg == "--init" {
			privKey := client.Signer()
			if privKey == nil {
				log.Fatal("--init requires STORM_PRIVATE_KEY to register public key")
			}
			pubKey := privKey.Public().(ed25519.PublicKey)
			opts = append(opts, storm.WithInit(), storm.WithPublicKey(pubKey))
		}
	}

	if err := client.Deposit(ctx, asset, &amount, opts...); err != nil {
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

	dir := parseDirection(dirStr)

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

	amount := tlb.MustFromDecimal(amountStr, 9)

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
	printResult(resp)
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

func printResult(resp *storm.PlaceOrderResult) {
	fmt.Printf("OK: %v\n", resp.Response.OK)
	fmt.Printf("  hash=%x\n", resp.OrderHash)
	if resp.StopLossHash != nil {
		fmt.Printf("  stopLoss hash=%x\n", resp.StopLossHash)
	}
	if resp.TakeProfitHash != nil {
		fmt.Printf("  takeProfit hash=%x\n", resp.TakeProfitHash)
	}
}

func cmdCloseAll(ctx context.Context) {
	if len(os.Args) < 5 {
		log.Fatal("usage: close-all <asset> <settlement> <long|short>")
	}

	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])

	resp, err := client.ClosePositionFull(ctx, market, dir)
	if err != nil {
		log.Fatalf("close position: %v", err)
	}
	printResult(resp)
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

	resp, err := client.ClosePosition(ctx, market, dir, &size)
	if err != nil {
		log.Fatalf("close position: %v", err)
	}
	printResult(resp)
}

func cmdStopLoss(ctx context.Context) {
	if len(os.Args) < 7 {
		log.Fatal("usage: stop-loss <asset> <settlement> <long|short> <amount> <triggerPrice>")
	}

	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	amount := tlb.MustFromDecimal(os.Args[5], 9)
	trigger := tlb.MustFromTON(os.Args[6])

	resp, err := client.PlaceStopLoss(ctx, market, dir, &amount, &trigger)
	if err != nil {
		log.Fatalf("place stop loss: %v", err)
	}
	printResult(resp)
}

func cmdTakeProfit(ctx context.Context) {
	if len(os.Args) < 7 {
		log.Fatal("usage: take-profit <asset> <settlement> <long|short> <amount> <triggerPrice>")
	}

	client := mustClient(ctx)
	market, err := client.Market(os.Args[2], os.Args[3])
	if err != nil {
		log.Fatalf("get market: %v", err)
	}
	dir := parseDirection(os.Args[4])
	amount := tlb.MustFromDecimal(os.Args[5], 9)
	trigger := tlb.MustFromTON(os.Args[6])

	resp, err := client.PlaceTakeProfit(ctx, market, dir, &amount, &trigger)
	if err != nil {
		log.Fatalf("place take profit: %v", err)
	}
	printResult(resp)
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
	dir := parseDirection(os.Args[4])
	amount := tlb.MustFromDecimal(os.Args[5], 9)

	resp, err := client.AddMargin(ctx, market, dir, &amount)
	if err != nil {
		log.Fatalf("add margin: %v", err)
	}
	printResult(resp)
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

	resp, err := client.RemoveMargin(ctx, market, dir, &amount)
	if err != nil {
		log.Fatalf("remove margin: %v", err)
	}
	printResult(resp)
}

func cmdInfo(ctx context.Context) {
	if len(os.Args) < 4 {
		log.Fatal("usage: info <asset> <settlement>")
	}

	assetName := os.Args[2]
	settlement := os.Args[3]

	client := mustClient(ctx)
	market, err := client.Market(assetName, settlement)
	if err != nil {
		log.Fatalf("get market: %v", err)
	}

	fmt.Printf("=== Market: %s (%s) ===\n\n", market.Ticker, market.Type)

	spotPrice, err := client.GetSpotPrice(ctx, market)
	if err != nil {
		fmt.Printf("[GetSpotPrice] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetSpotPrice] %s\n", spotPrice)
	}

	termPrice, err := client.GetTerminalAmmPrice(ctx, market)
	if err != nil {
		fmt.Printf("[GetTerminalAmmPrice] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetTerminalAmmPrice] %s\n", termPrice)
	}

	status, err := client.GetAmmStatus(ctx, market)
	if err != nil {
		fmt.Printf("[GetAmmStatus] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetAmmStatus] closeOnly=%v paused=%v stopped=%v\n", status.CloseOnly, status.Paused, status.Stopped)
	}

	state, err := client.GetAmmState(ctx, market)
	if err != nil {
		fmt.Printf("[GetAmmState] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetAmmState] quoteReserve=%s baseReserve=%s weight=%d\n", state.QuoteAssetReserve, state.BaseAssetReserve, state.QuoteAssetWeight)
		fmt.Printf("  longSize=%s shortSize=%s\n", state.TotalLongPositionSize, state.TotalShortPositionSize)
		fmt.Printf("  OI long=%s short=%s\n", state.OpenInterestLong, state.OpenInterestShort)
		fmt.Printf("  nextFundingTS=%d\n", state.NextFundingBlockTimestamp)
	}

	oracle, err := client.GetOracleData(ctx, market)
	if err != nil {
		fmt.Printf("[GetOracleData] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetOracleData] price=%s spread=%s timestamp=%d\n", oracle.OracleLastPrice, oracle.OracleLastSpread, oracle.OracleLastTimestamp)
	}

	settings, err := client.GetExchangeSettings(ctx, market)
	if err != nil {
		fmt.Printf("[GetExchangeSettings] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetExchangeSettings] fee=%d rolloverFee=%d fundingPeriod=%d\n", settings.Fee, settings.RolloverFee, settings.FundingPeriod)
		fmt.Printf("  initMargin=%d maintMargin=%d liqFeeRatio=%d\n", settings.InitMarginRatio, settings.MaintenanceMarginRatio, settings.LiquidationFeeRatio)
		fmt.Printf("  maxOpenNotional=%s spreadLimit=%d\n", settings.MaxOpenNotional, settings.SpreadLimit)
	}

	dayTrading, err := client.GetDayTradingData(ctx, market)
	if err != nil {
		fmt.Printf("[GetDayTradingData] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetDayTradingData] active=%v maxLeverage=%s\n", dayTrading.Active, dayTrading.MaxLeverage)
	}

	if oracle != nil && oracle.OracleLastPrice != nil {
		oraclePrice := oracle.OracleLastPrice.Nano()
		decimalUnit := big.NewInt(1_000_000_000)
		funding, err := client.GetFunding(ctx, market, oraclePrice, decimalUnit)
		if err != nil {
			fmt.Printf("[GetFunding] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetFunding] long=%s short=%s premiumToVault=%s\n", funding.LongFunding, funding.ShortFunding, funding.PremiumToVault)
		}
	}

	if oracle != nil && oracle.OracleLastPrice != nil {
		oraclePrice := oracle.OracleLastPrice.Nano()
		premium, err := client.GetPremium(ctx, market, oraclePrice)
		if err != nil {
			fmt.Printf("[GetPremium] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetPremium] %s\n", premium)
		}
	}

	if oracle != nil && oracle.OracleLastPrice != nil {
		oraclePrice := oracle.OracleLastPrice.Nano()
		decimalUnit := big.NewInt(1_000_000_000)
		fakePos := &stlb.PositionState{
			Size:         big.NewInt(1_000_000_000),
			Direction:    0,
			Margin:       coinFromNano(100_000_000_000),
			OpenNotional: coinFromNano(1_000_000_000_000),
		}
		posCell, err := tlb.ToCell(fakePos)
		if err != nil {
			fmt.Printf("[GetRemainMargin] serialize error: %v\n", err)
		} else {
			margin, err := client.GetRemainMargin(ctx, market, oraclePrice, posCell, decimalUnit)
			if err != nil {
				fmt.Printf("[GetRemainMargin] ERROR: %v\n", err)
			} else {
				fmt.Printf("[GetRemainMargin] remain=%s funding=%s marginRatio=%s pnl=%s\n", margin.RemainMargin, margin.FundingPayment, margin.MarginRatio, margin.UnrealizedPnl)
			}
		}
	}

	saAddr := client.SmartAccount()
	tonAPI := client.TonAPI()
	if saAddr != nil && tonAPI != nil {
		vc := vammclient.NewClient(tonAPI, market.VammAddress)
		pmAddr, err := vc.GetPositionManagerAddress(ctx, saAddr)
		if err != nil {
			fmt.Printf("[GetPositionManagerAddress] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetPositionManagerAddress] %s\n", pmAddr)
		}
	}

	fmt.Println()

	fmt.Printf("=== Vault: %s ===\n\n", settlement)

	vaultData, err := client.GetVaultData(ctx, settlement)
	if err != nil {
		fmt.Printf("[GetVaultData] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetVaultData] rate=%s lpSupply=%s free=%s locked=%s\n", vaultData.Rate, vaultData.LpTotalSupply, vaultData.FreeBalance, vaultData.LockedBalance)
		fmt.Printf("  buffer=%s stakers=%s executors=%s v3Paused=%v\n", vaultData.BufferBalance, vaultData.StakersBalance, vaultData.ExecutorsBalance, vaultData.V3Paused)
	}

	bufferData, err := client.GetBufferData(ctx, settlement)
	if err != nil {
		fmt.Printf("[GetBufferData] ERROR: %v\n", err)
	} else {
		fmt.Printf("[GetBufferData] balance=%s rate=%s underRate=%s overRate=%s\n", bufferData.Balance, bufferData.Rate, bufferData.UnderRate, bufferData.OverRate)
	}

	asset, _ := client.Asset(settlement)
	if asset != nil && tonAPI != nil {
		vc := vaultclient.NewClient(tonAPI, asset.VaultAddress)

		lpMinter, err := vc.GetLpMinterAddress(ctx)
		if err != nil {
			fmt.Printf("[GetLpMinterAddress] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetLpMinterAddress] %s\n", lpMinter)
		}

		vammAddr, err := vc.GetVAMMAddress(ctx, market.AssetIndex)
		if err != nil {
			fmt.Printf("[GetVAMMAddress(%d)] ERROR: %v\n", market.AssetIndex, err)
		} else {
			fmt.Printf("[GetVAMMAddress(%d)] %s\n", market.AssetIndex, vammAddr)
		}

		if saAddr != nil {
			posAddr, err := vc.GetPositionAddress(ctx, saAddr, market.VammAddress)
			if err != nil {
				fmt.Printf("[GetPositionAddress] ERROR: %v\n", err)
			} else {
				fmt.Printf("[GetPositionAddress] %s\n", posAddr)
			}
		}
	}

	if tonAPI != nil {
		fmt.Println()
		fmt.Printf("=== Factory ===\n\n")

		factory := smartaccount.NewFactory(tonAPI, address.MustParseAddr(config.Networks[client.Network()].FactoryAddress))

		factoryData, err := factory.GetFactoryData(ctx)
		if err != nil {
			fmt.Printf("[GetFactoryData] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetFactoryData] admin=%s timeout=%d version=%d\n", factoryData.AdminAddress, factoryData.HighloadTimeout, factoryData.Version)
		}

		minFees, err := factory.GetMinFees(ctx)
		if err != nil {
			fmt.Printf("[GetMinFees] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetMinFees] depositNative=%s depositJetton=%s deployNative=%s deployJetton=%s withdraw=%s storage=%s\n",
				minFees.DepositMinFeeNative, minFees.DepositMinFeeJetton,
				minFees.DepositWithDeployMinFeeNative, minFees.DepositWithDeployMinFeeJetton,
				minFees.WithdrawMinFee, minFees.StorageFee)
		}
	}

	if saAddr != nil && tonAPI != nil {
		fmt.Println()
		fmt.Printf("=== Smart Account: %s ===\n\n", saAddr)

		saClient := smartaccount.NewClient(tonAPI, saAddr)

		nftData, err := saClient.GetNftData(ctx)
		if err != nil {
			fmt.Printf("[GetNftData] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetNftData] init=%v index=%s owner=%s collection=%s\n", nftData.Init, nftData.Index, nftData.OwnerAddress, nftData.CollectionAddress)
		}

		keysData, err := saClient.GetKeysData(ctx)
		if err != nil {
			fmt.Printf("[GetKeysData] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetKeysData] keysCount=%d hasUserKeys=%v\n", keysData.KeysCount, keysData.UserKeys != nil)
		}

		highloadData, err := saClient.GetHighloadData(ctx)
		if err != nil {
			fmt.Printf("[GetHighloadData] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetHighloadData] lastCleanTime=%d timeout=%d\n", highloadData.LastCleanTime, highloadData.Timeout)
		}

		if asset != nil {
			balance, err := saClient.GetBalance(ctx, asset.VaultAddress)
			if err != nil {
				fmt.Printf("[GetBalance(%s)] ERROR: %v\n", settlement, err)
			} else {
				fmt.Printf("[GetBalance(%s)] %s\n", settlement, balance)
			}
		}

		pubKeys, err := saClient.GetUserPublicKeys(ctx)
		if err != nil {
			fmt.Printf("[GetUserPublicKeys] ERROR: %v\n", err)
		} else {
			fmt.Printf("[GetUserPublicKeys] hasKeys=%v\n", pubKeys != nil)
		}

		pos, err := saClient.GetPosition(ctx, market.VammAddress, 0)
		if err != nil {
			fmt.Printf("[GetPosition(long)] ERROR: %v\n", err)
		} else if pos == nil {
			fmt.Printf("[GetPosition(long)] no position\n")
		} else {
			fmt.Printf("[GetPosition(long)] locked=%v hasData=%v\n", pos.Locked, pos.Data != nil)
		}

		pos, err = saClient.GetPosition(ctx, market.VammAddress, 1)
		if err != nil {
			fmt.Printf("[GetPosition(short)] ERROR: %v\n", err)
		} else if pos == nil {
			fmt.Printf("[GetPosition(short)] no position\n")
		} else {
			fmt.Printf("[GetPosition(short)] locked=%v hasData=%v\n", pos.Locked, pos.Data != nil)
		}
	}

	if saAddr != nil && tonAPI != nil {
		fmt.Println()
		fmt.Printf("=== Position Manager ===\n\n")

		vc := vammclient.NewClient(tonAPI, market.VammAddress)
		pmAddr, err := vc.GetPositionManagerAddress(ctx, saAddr)
		if err != nil {
			fmt.Printf("[GetPositionManagerAddress] ERROR: %v\n", err)
		} else {
			pm := pmclient.NewClient(tonAPI, pmAddr)

			inited, err := pm.GetIsInited(ctx)
			if err != nil {
				fmt.Printf("[GetIsInited] ERROR: %v\n", err)
			} else {
				fmt.Printf("[GetIsInited] %v\n", inited)
			}

			if inited {
				version, err := pm.GetVersion(ctx)
				if err != nil {
					fmt.Printf("[GetVersion] ERROR: %v\n", err)
				} else {
					fmt.Printf("[GetVersion] %d\n", version)
				}

				pmData, err := pm.GetPositionManagerData(ctx)
				if err != nil {
					fmt.Printf("[GetPositionManagerData] ERROR: %v\n", err)
				} else {
					fmt.Printf("[GetPositionManagerData] trader=%s vault=%s vamm=%s\n", pmData.TraderAddress, pmData.VaultAddress, pmData.VammAddress)
					fmt.Printf("  hasLong=%v hasShort=%v hasOrders=%v ordersBitset=%d\n",
						pmData.LongPosition != nil, pmData.ShortPosition != nil, pmData.Orders != nil, pmData.OrdersBitset)
				}
			}
		}
	}
}

func coinFromNano(n int64) *tlb.Coins {
	c := tlb.FromNanoTON(big.NewInt(n))
	return &c
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
