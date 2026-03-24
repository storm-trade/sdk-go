package vamm

import (
	"github.com/xssnick/tonutils-go/address"
	"github.com/xssnick/tonutils-go/tlb"
	"github.com/xssnick/tonutils-go/tvm/cell"
)

type OracleData struct {
	OracleLastPrice      *tlb.Coins `tlb:"." json:"oracle_last_price"`
	OracleLastSpread     *tlb.Coins `tlb:"." json:"oracle_last_spread"`
	OracleLastTimestamp  uint32     `tlb:"## 32" json:"oracle_last_timestamp"`
	OracleMaxDeviation   *tlb.Coins `tlb:"." json:"oracle_max_deviation"`
	OracleValidityPeriod uint32     `tlb:"## 32" json:"oracle_validity_period"`
	OracleAssetID        uint16     `tlb:"## 16" json:"oracle_asset_id"`
}

type AmmData struct {
	VaultAddress          *address.Address  `tlb:"addr" json:"vault_address"`
	AssetId               uint16            `tlb:"## 16" json:"asset_id"`
	CloseOnly             bool              `tlb:"bool" json:"close_only"`
	Paused                bool              `tlb:"bool" json:"paused"`
	OracleLastPrice       *tlb.Coins        `tlb:"." json:"oracle_last_price"`
	OracleLastSpread      *tlb.Coins        `tlb:"." json:"oracle_last_spread"`
	OracleLastTimestamp   uint32            `tlb:"## 32" json:"oracle_last_timestamp"`
	OracleMaxDeviation    *tlb.Coins        `tlb:"." json:"oracle_max_deviation"`
	OracleValidityPeriod  uint32            `tlb:"## 32" json:"oracle_validity_period"`
	OraclePublicKeysCount uint32            `tlb:"## 4" json:"oracle_public_keys_count"`
	AmmState              *AmmState         `tlb:"^" json:"amm_state"`
	Settings              *ExchangeSettings `tlb:"^" json:"settings"`
	PauseAt               uint32            `tlb:"-" json:"pause_at"`
	UnpauseAt             uint32            `tlb:"-" json:"unpause_at"`
	SettlementOracleData  *OracleData       `tlb:"-" json:"settlement_oracle_data,omitempty"`
}

type AmmState struct {
	QuoteAssetReserve                    *tlb.Coins `tlb:"." json:"quote_asset_reserve"`
	BaseAssetReserve                     *tlb.Coins `tlb:"." json:"base_asset_reserve"`
	QuoteAssetWeight                     uint64     `tlb:"## 64" json:"quote_asset_reserve_weight"`
	TotalLongPositionSize                *tlb.Coins `tlb:"." json:"total_long_position_size"`
	TotalShortPositionSize               *tlb.Coins `tlb:"." json:"total_short_position_size"`
	OpenInterestLong                     *tlb.Coins `tlb:"." json:"open_interest_long"`
	OpenInterestShort                    *tlb.Coins `tlb:"." json:"open_interest_short"`
	LatestLongCumulativePremiumFraction  int64      `tlb:"## 64" json:"latest_long_cumulative_premium_fraction"`
	LatestShortCumulativePremiumFraction int64      `tlb:"## 64" json:"latest_short_cumulative_premium_fraction"`
	NextFundingBlockTimestamp            uint32     `tlb:"## 32" json:"next_funding_block_timestamp"`
}

type CommonExchangeSettings struct {
	Fee                           uint32           `tlb:"## 32" json:"fee"`
	RolloverFee                   uint32           `tlb:"## 32" json:"rollover_fee"`
	FundingPeriod                 uint32           `tlb:"## 32" json:"funding_period"`
	InitMarginRatio               uint32           `tlb:"## 32" json:"init_margin_ratio"`
	MaintenanceMarginRatio        uint32           `tlb:"## 32" json:"maintenance_margin_ratio"`
	LiquidationFeeRatio           uint32           `tlb:"## 32" json:"liquidation_fee_ratio"`
	PartialLiquidationRatio       uint32           `tlb:"## 32" json:"partial_liquidation_ratio"`
	SpreadLimit                   uint32           `tlb:"## 32" json:"spread_limit"`
	MaxPriceImpact                uint32           `tlb:"## 32" json:"max_price_impact"`
	MaxPriceSpread                uint32           `tlb:"## 32" json:"max_price_spread"`
	MaxOpenNotional               *tlb.Coins       `tlb:"." json:"max_open_notional"`
	FeeToStakersPercent           uint32           `tlb:"## 32" json:"fee_to_stakers_percent"`
	FundingMode                   uint8            `tlb:"## 2" json:"funding_mode"`
	MinPartialLiquidationNotional *tlb.Coins       `tlb:"." json:"min_partial_liquidation_notional"`
	MinLeverage                   uint32           `tlb:"## 32" json:"min_leverage"`
	DirectIncreaseEnabled         bool             `tlb:"bool" json:"direct_increase_enabled"`
	DirectCloseEnabled            bool             `tlb:"bool" json:"direct_close_enabled"`
	WhitelistAddresses            *cell.Dictionary `tlb:"dict 4" json:"whitelist_addresses"`
}

type FundingSettings struct {
	LowFundingFnA   int64 `tlb:"## 64" json:"low_funding_fn_a"`
	LowFundingFnB   int64 `tlb:"## 64" json:"low_funding_fn_b"`
	HighFundingFnA  int64 `tlb:"## 64" json:"high_funding_fn_a"`
	HighFundingFnB  int64 `tlb:"## 64" json:"high_funding_fn_b"`
	InflectionPoint int64 `tlb:"## 64" json:"inflection_point"`
}

type ExchangeSettings struct {
	*CommonExchangeSettings
	*FundingSettings
}
