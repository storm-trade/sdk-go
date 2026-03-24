package vamm

import "math/big"

type AmmStatus struct {
	CloseOnly bool
	Paused    bool
	Stopped   bool
	PauseAt   uint32
	UnpauseAt uint32
}

type FundingData struct {
	LongFunding        *big.Int
	ShortFunding       *big.Int
	PremiumToVault     *big.Int
	SyncExchangeAmount *big.Int
	SpotPrice          *big.Int
}

type DayTradingData struct {
	Active      bool
	MaxLeverage *big.Int
}

type MarginData struct {
	RemainMargin     *big.Int
	FundingPayment   *big.Int
	MarginRatio      *big.Int
	UnrealizedPnl    *big.Int
	BadDebt          *big.Int
	PositionNotional *big.Int
	RolloverFee      *big.Int
	OraclePrice      *big.Int
	SpotPrice        *big.Int
}
