package storm

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/storm-trade/sdk-go/config"
	sacontracts "github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton/jetton"
	"github.com/xssnick/tonutils-go/ton/wallet"
)

var gaslessWallets = map[config.Network]*address.Address{
	config.Testnet: address.MustParseAddr("kQDgjK1GCrTL7J-VZKoaISNU599e0o47KcIuQi1G2Ll2IUap"),
}

var gaslessAssetAddresses = map[config.Network]map[string]*address.Address{
	config.Testnet: {
		"USDT": address.MustParseAddr("kQBA63OEBglSTatm3tTForJ8bjgv7VMGNi2-hNMKeONZVBAr"),
	},
}

func (c *Client) GaslessDeposit(ctx context.Context, assetName string, amount *tlb.Coins, opts ...Option) error {
	o := c.resolveOptions(opts)
	if o.tonAPI == nil {
		return fmt.Errorf("TON API required: use WithTONApi()")
	}
	if o.wallet == nil {
		return fmt.Errorf("wallet required: use WithWallet()")
	}

	gaslessWallet := gaslessWallets[c.network]
	if gaslessWallet == nil {
		return fmt.Errorf("gasless not available for this network")
	}

	asset, err := c.Asset(assetName)
	if err != nil {
		return err
	}

	if asset.JettonMaster != nil {
		jw, err := jetton.NewJettonMasterClient(o.tonAPI, asset.JettonMaster).GetJettonWallet(ctx, o.wallet.WalletAddress())
		if err != nil {
			return fmt.Errorf("get jetton wallet: %w", err)
		}

		fwd := tlb.FromNanoTONU(1)
		payload, err := jetton.BuildTransferPayload(gaslessWallet, o.wallet.WalletAddress(), *amount, fwd, nil, nil)
		if err != nil {
			return fmt.Errorf("build jetton transfer: %w", err)
		}

		gas := tlb.MustFromTON("0.05")
		msg := wallet.SimpleMessage(jw.Address(), gas, payload)
		_, _, err = o.wallet.SendWaitTransaction(ctx, msg)
		return err
	}

	msg := wallet.SimpleMessage(gaslessWallet, *amount, nil)
	_, _, err = o.wallet.SendWaitTransaction(ctx, msg)
	return err
}

func (c *Client) GetGaslessBalance(ctx context.Context, opts ...Option) (*sequencer.GaslessBalance, error) {
	o := c.resolveOptions(opts)
	if o.wallet == nil {
		return nil, fmt.Errorf("wallet required: use WithWallet()")
	}
	return c.seq.GetGaslessBalance(ctx, o.wallet.WalletAddress().String())
}

func (c *Client) GetGaslessWithdrawals(ctx context.Context, opts ...Option) ([]sequencer.GaslessWithdrawalEntry, error) {
	o := c.resolveOptions(opts)
	if o.wallet == nil {
		return nil, fmt.Errorf("wallet required: use WithWallet()")
	}
	return c.seq.GetGaslessWithdrawals(ctx, o.wallet.WalletAddress().String())
}

func (c *Client) GaslessWithdraw(ctx context.Context, assetName string, amount *tlb.Coins, opts ...Option) (*sequencer.GaslessWithdrawResponse, error) {
	o := c.resolveOptions(opts)
	if o.signer == nil {
		return nil, fmt.Errorf("signer required: use WithSigner()")
	}
	if o.smartAccount == nil {
		return nil, fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	var assetAddr *address.Address
	if assetName != config.NativeAssetID {
		assets := gaslessAssetAddresses[c.network]
		if assets == nil {
			return nil, fmt.Errorf("gasless not available for this network")
		}
		addr, ok := assets[assetName]
		if !ok {
			return nil, fmt.Errorf("gasless withdrawal not supported for asset %q", assetName)
		}
		assetAddr = addr
	}

	nonce := generateNonce()

	payload := &sacontracts.WithdrawalPayload{
		SmartAccount: o.smartAccount,
		AssetAddress: assetAddr,
		Amount:       amount,
		Nonce:        nonce,
		CreatedAt:    uint64(time.Now().Add(-o.clockSkew).Unix()),
	}

	payloadCell, err := tlb.ToCell(payload)
	if err != nil {
		return nil, fmt.Errorf("serialize withdrawal payload: %w", err)
	}

	signature := payloadCell.Sign(o.signer)
	pubKey := []byte(o.signer.Public().(ed25519.PublicKey))

	return c.seq.GaslessWithdraw(ctx, sequencer.GaslessWithdrawRequest{
		SmartAccount: o.smartAccount.String(),
		Message:      base64.StdEncoding.EncodeToString(payloadCell.ToBOC()),
		PublicKey:    base64.StdEncoding.EncodeToString(pubKey),
		Signature:    base64.StdEncoding.EncodeToString(signature),
	})
}

func generateNonce() uint64 {
	var b [8]byte
	rand.Read(b[:])
	return binary.BigEndian.Uint64(b[:])
}
