package storm

import (
	"context"
	"crypto/ed25519"
	"fmt"

	sa "github.com/storm-trade/sdk-go/client/smartaccount"
	sacontracts "github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/tlb"
)

func (c *Client) Deposit(ctx context.Context, assetName string, amount *tlb.Coins, opts ...Option) error {
	o := c.resolveOptions(opts)

	if o.tonAPI == nil {
		return fmt.Errorf("TON API required for deposit: use WithTONApi()")
	}
	if o.wallet == nil {
		return fmt.Errorf("wallet required for deposit: use WithWallet()")
	}
	if o.smartAccount == nil {
		return fmt.Errorf("smart account required for deposit: use WithSmartAccount()")
	}

	if o.publicKey != nil && !o.init {
		return fmt.Errorf("public key can only be set with WithInit(): contract rejects keyInit without init")
	}

	asset, err := c.Asset(assetName)
	if err != nil {
		return err
	}

	saClient := sa.NewClient(o.tonAPI, o.smartAccount)

	var pubKeys []sacontracts.PublicKey
	if o.publicKey != nil {
		pubKeys = append(pubKeys, sacontracts.PublicKey(o.publicKey))
	}

	if asset.JettonMaster != nil {
		_, err = saClient.DepositJetton(o.wallet, o.wallet.WalletAddress(), asset.VaultAddress, asset.JettonMaster, amount, o.init, pubKeys...)
	} else {
		_, err = saClient.DepositNative(o.wallet, o.wallet.WalletAddress(), asset.VaultAddress, amount, o.init, pubKeys...)
	}

	return err
}

func (c *Client) Withdraw(ctx context.Context, assetName string, amount *tlb.Coins, opts ...Option) error {
	o := c.resolveOptions(opts)

	if o.tonAPI == nil {
		return fmt.Errorf("TON API required for withdraw: use WithTONApi()")
	}
	if o.wallet == nil {
		return fmt.Errorf("wallet required for withdraw: use WithWallet()")
	}
	if o.smartAccount == nil {
		return fmt.Errorf("smart account required for withdraw: use WithSmartAccount()")
	}

	asset, err := c.Asset(assetName)
	if err != nil {
		return err
	}

	saClient := sa.NewClient(o.tonAPI, o.smartAccount)
	_, err = saClient.Withdraw(o.wallet, asset.VaultAddress, amount)
	return err
}

func (c *Client) AddPublicKey(ctx context.Context, opts ...Option) error {
	o := c.resolveOptions(opts)

	if o.tonAPI == nil {
		return fmt.Errorf("TON API required for add-key: use WithTONApi()")
	}
	if o.wallet == nil {
		return fmt.Errorf("wallet required for add-key: use WithWallet()")
	}
	if o.smartAccount == nil {
		return fmt.Errorf("smart account required for add-key: use WithSmartAccount()")
	}
	if o.signer == nil {
		return fmt.Errorf("signer required for add-key: use WithSigner()")
	}

	saClient := sa.NewClient(o.tonAPI, o.smartAccount)
	pubKey := sacontracts.PublicKey(o.signer.Public().(ed25519.PublicKey))
	_, err := saClient.AddPublicKey(o.wallet, pubKey)
	return err
}

func (c *Client) GetBalances(ctx context.Context, opts ...Option) (map[string]int64, error) {
	o := c.resolveOptions(opts)
	if o.smartAccount == nil {
		return nil, fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	return c.seq.GetBalances(ctx, o.smartAccount.String())
}

func (c *Client) GetIntent(ctx context.Context, hash string) (*sequencer.SettledAction, error) {
	return c.seq.GetIntent(ctx, hash)
}

func (c *Client) GetPositions(ctx context.Context, opts ...Option) ([]sequencer.PositionResponse, error) {
	o := c.resolveOptions(opts)
	if o.smartAccount == nil {
		return nil, fmt.Errorf("smart account required: use WithSmartAccount()")
	}

	return c.seq.GetPositions(ctx, o.smartAccount.String())
}
