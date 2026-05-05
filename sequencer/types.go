package sequencer

import "encoding/json"

type PaymentMode string

const (
	PaymentModeDefault PaymentMode = ""
	PaymentModeGasless PaymentMode = "gasless"
)

type Status struct {
	OK                 bool   `json:"ok"`
	Lag                uint64 `json:"lag"`
	LastProcessedBlock uint32 `json:"last_processed_block"`
	CurrentBlock       uint32 `json:"current_block"`
	ActiveShards       int    `json:"active_shards"`
	PendingBundles     int    `json:"pending_bundles"`
	FinalizingBundles  int    `json:"finalizing_bundles"`
	CommittedBundles   int    `json:"committed_bundles"`
}

type AccountState struct {
	Type      int                      `json:"type"`
	Factory   string                   `json:"factory"`
	Owner     string                   `json:"owner"`
	Balances  []AccountBalance         `json:"balances"`
	Version   int                      `json:"version"`
	Keys      *AccountKeys             `json:"keys,omitempty"`
	Positions map[string]PositionEntry `json:"positions,omitempty"`
}

type AccountBalance struct {
	Address string `json:"address"`
	Amount  int64  `json:"amount"`
}

type AccountKeys struct {
	Hot            string          `json:"hot"`
	Cold           string          `json:"cold"`
	UserPublicKeys *UserPublicKeys `json:"user_public_keys,omitempty"`
	KeysCount      int             `json:"keys_count"`
}

type UserPublicKeys struct {
	Values []string `json:"values"`
}

type BalanceResponse struct {
	MinterAddress string `json:"minter_address,omitempty"`
	Amount        uint64 `json:"amount"`
}

type PositionEntry struct {
	Locked bool          `json:"locked"`
	Data   *PositionData `json:"data,omitempty"`
}

type PositionData struct {
	Size                         int64  `json:"size"`
	Direction                    int    `json:"direction"`
	Margin                       string `json:"margin"`
	OpenNotional                 string `json:"open_notional"`
	LastUpdatedCumulativePremium int64  `json:"last_updated_cumulative_premium"`
	Fee                          uint64 `json:"fee"`
	Discount                     uint64 `json:"discount"`
	Rebate                       uint64 `json:"rebate"`
	LastUpdatedTimestamp         uint64 `json:"last_updated_timestamp"`
}

type PositionResponse struct {
	Market    string                 `json:"market"`
	Direction string                 `json:"direction"`
	State     *PositionStateResponse `json:"state"`
}

type PositionStateResponse struct {
	Status   string        `json:"status"`
	Value    *PositionData `json:"value"`
	BundleID *string       `json:"bundle_id,omitempty"`
}

type PlaceOrderRequest struct {
	SmartAccount  string               `json:"sa"`
	Message       string               `json:"message"`
	PublicKey     string               `json:"public_key"`
	Signature     string               `json:"signature"`
	OrderRequests []SignedOrderRequest `json:"order_requests,omitempty"`
	PaymentMode   PaymentMode          `json:"payment_mode,omitempty"`
}

type SignedOrderRequest struct {
	Message   string `json:"message"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type PlaceOrderResponse struct {
	OK     bool           `json:"ok"`
	Trace  *TraceResult   `json:"trace,omitempty"`
	Intent *SettledAction `json:"intent,omitempty"`
}

type TraceResult struct {
	ExitCode               int    `json:"exit_code"`
	ContractExitCode       int    `json:"contract_exit_code"`
	ErrorContractInterface string `json:"error_contract_interface,omitempty"`
}

type SettledAction struct {
	Hash          string          `json:"hash"`
	Date          string          `json:"date,omitempty"`
	ActionPhase   string          `json:"action_phase,omitempty"`
	Market        string          `json:"market,omitempty"`
	Vault         string          `json:"vault,omitempty"`
	SmartAccount  string          `json:"smart_account,omitempty"`
	PaymentMode   uint8           `json:"payment_mode,omitempty"`
	Price         json.RawMessage `json:"price,omitempty"`
	CreatedPrice  json.RawMessage `json:"created_price,omitempty"`
	Order         json.RawMessage `json:"order,omitempty"`
	FullMessage   json.RawMessage `json:"full_message,omitempty"`
	OrderRequests json.RawMessage `json:"order_requests,omitempty"`
}

type SyncPositionRequest struct {
	SmartAccount string `json:"smart_account"`
	VAmm         string `json:"vamm"`
	Direction    string `json:"direction"`
}

type CancelOrderRequest struct {
	SmartAccount string `json:"sa"`
	Message      string `json:"message"`
	PublicKey    string `json:"public_key"`
	Signature    string `json:"signature"`
}

type OrderbookDepth struct {
	Bids []DepthLevel `json:"bids"`
	Asks []DepthLevel `json:"asks"`
}

type DepthLevel struct {
	Price int64 `json:"price"`
	Count int   `json:"count"`
}

type OrderbookOrder struct {
	Key          string          `json:"key"`
	Price        int64           `json:"price"`
	SmartAccount string          `json:"smart_account"`
	VAmm         string          `json:"vamm"`
	Order        json.RawMessage `json:"order"`
}

type EventsFilter struct {
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Order     string `json:"order,omitempty"`
}

type TxStatus struct {
	Success          bool `json:"success"`
	ExitCode         int  `json:"exit_code"`
	ActionResultCode int  `json:"action_result_code"`
}

type Event struct {
	Sequence  uint64          `json:"sequence"`
	Timestamp int64           `json:"timestamp"`
	Hash      string          `json:"hash"`
	Account   string          `json:"account"`
	TxStatus  *TxStatus       `json:"tx_status,omitempty"`
	Sender    string          `json:"sender"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}

type BundleID struct {
	Addr      string `json:"addr"`
	Market    string `json:"market"`
	Direction string `json:"direction"`
	Owner     string `json:"owner"`
}

type Bundle struct {
	ID            BundleID        `json:"id"`
	LeasedQueryID uint64          `json:"leased_query_id"`
	CreatedAt     string          `json:"created_at"`
	LastUpdate    string          `json:"last_update"`
	Intents       json.RawMessage `json:"intents,omitempty"`
	IntentHashes  []string        `json:"intent_hashes"`
	PositionState json.RawMessage `json:"position_state,omitempty"`
	LatestTrace   *TraceResult    `json:"latest_trace,omitempty"`
	ExtHash       json.RawMessage `json:"ext_hash,omitempty"`
	TxHash        json.RawMessage `json:"tx_hash,omitempty"`
	PaymentMode   uint8           `json:"payment_mode"`
	GaslessSource uint8           `json:"gasless_source"`
}

type GaslessBalance struct {
	Balances  map[string]string `json:"balances"`
	UpdatedAt string            `json:"updated_at"`
}

type GaslessWithdrawalEntry struct {
	AssetAddress string `json:"asset_address"`
	Amount       string `json:"amount"`
	Status       string `json:"status"`
	MessageHash  string `json:"message_hash"`
	CreatedAt    string `json:"created_at"`
	SentAt       string `json:"sent_at,omitempty"`
	ExecutedAt   string `json:"executed_at,omitempty"`
}

type GaslessWithdrawRequest struct {
	SmartAccount string `json:"sa"`
	Message      string `json:"message"`
	PublicKey    string `json:"public_key"`
	Signature    string `json:"signature"`
}

type GaslessWithdrawResponse struct {
	OK           bool            `json:"ok"`
	WithdrawalID json.RawMessage `json:"withdrawal_id"`
	Status       string          `json:"status"`
	TxHash       string          `json:"tx_hash"`
	Amount       string          `json:"amount"`
	Asset        string          `json:"asset"`
	Destination  string          `json:"destination"`
	QueryID      uint64          `json:"query_id"`
}
