package storm

import (
	"context"
	"crypto/ed25519"
	"errors"
	"fmt"

	sa "github.com/storm-trade/sdk-go/client/smartaccount"
	vaultclient "github.com/storm-trade/sdk-go/client/vault"
	"github.com/storm-trade/sdk-go/config"
	sacontracts "github.com/storm-trade/sdk-go/contracts/smartaccount"
	"github.com/storm-trade/sdk-go/sequencer"
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/ton"
)

var errZeroByteKey = errors.New("public key starts with 0x00, which the sequencer cannot use: generate another key")

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
	if o.publicKey != nil && o.publicKey[0] == 0 {
		return errZeroByteKey
	}

	asset, err := c.walletAsset(ctx, o, assetName)
	if err != nil {
		return err
	}

	saClient := sa.NewClient(o.tonAPI, o.smartAccount)

	nft, err := saClient.GetNftData(ctx)
	var execErr ton.ContractExecError
	if err != nil && !(errors.As(err, &execErr) && execErr.Code == ton.ErrCodeContractNotInitialized) {
		return fmt.Errorf("read smart account: %w", err)
	}
	initialized := err == nil && nft.Init
	if o.init && initialized {
		return fmt.Errorf("smart account %s is already initialized: deposit without WithInit()", o.smartAccount)
	}
	if o.publicKey != nil && initialized {
		return fmt.Errorf("smart account %s is already initialized: register the key with AddPublicKey()", o.smartAccount)
	}

	var pubKeys []sacontracts.PublicKey
	if o.publicKey != nil {
		pubKeys = append(pubKeys, sacontracts.PublicKey(o.publicKey))
	}

	if asset.JettonMaster != nil {
		_, err = saClient.DepositJetton(o.wallet, o.wallet.WalletAddress(), asset.VaultAddress, asset.JettonMaster, amount, !initialized, pubKeys...)
	} else {
		_, err = saClient.DepositNative(o.wallet, o.wallet.WalletAddress(), asset.VaultAddress, amount, !initialized, pubKeys...)
	}

	return err
}

func (c *Client) walletAsset(ctx context.Context, o clientOptions, assetName string) (*config.Asset, error) {
	asset, err := c.Asset(assetName)
	if err != nil {
		return nil, err
	}
	if config.LegacyAssets[asset.Name] {
		return nil, fmt.Errorf("asset %s is on a retired vault and cannot be used", asset.Name)
	}

	c.vaultMu.Lock()
	defer c.vaultMu.Unlock()
	if c.checkedVaults[asset.VaultAddress.StringRaw()] {
		return asset, nil
	}
	factory, err := vaultclient.NewClient(o.tonAPI, asset.VaultAddress).GetSmartAccountFactoryAddress(ctx)
	if err != nil {
		return nil, fmt.Errorf("read smart account factory of the %s vault: %w", asset.Name, err)
	}
	if sdkFactory := address.MustParseAddr(config.Networks[c.network].FactoryAddress); !factory.Equals(sdkFactory) {
		return nil, fmt.Errorf("the %s vault uses smart account factory %s, this SDK uses %s: update the SDK", asset.Name, factory, sdkFactory)
	}
	c.checkedVaults[asset.VaultAddress.StringRaw()] = true
	return asset, nil
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

	asset, err := c.walletAsset(ctx, o, assetName)
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
	if pubKey[0] == 0 {
		return errZeroByteKey
	}
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
