package oracle

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/xssnick/tonutils-go/tvm/cell"
)

const (
	TestnetURL = "https://oracle.stage.stormtrade.dev"
	MainnetURL = "https://oracle.storm.tg"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string) *Client {
	return &Client{baseURL: baseURL, httpClient: &http.Client{}}
}

type SignedPrice struct {
	PriceRef      *cell.Cell
	SignaturesRef *cell.Cell
}

type priceResponse struct {
	ResultMessage struct {
		PriceRef      string `json:"price_ref"`
		SignaturesRef string `json:"signatures_ref"`
	} `json:"result_message"`
}

func (c *Client) GetSignedPrice(ctx context.Context, symbol string) (*SignedPrice, error) {
	url := fmt.Sprintf("%s/v2/signed/%s", c.baseURL, symbol)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch oracle price: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read oracle response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oracle API returned %d: %s", resp.StatusCode, string(body[:min(len(body), 200)]))
	}

	var pr priceResponse
	if err := json.Unmarshal(body, &pr); err != nil {
		return nil, fmt.Errorf("parse oracle response: %w", err)
	}

	priceRef, err := cellFromBase64(pr.ResultMessage.PriceRef)
	if err != nil {
		return nil, fmt.Errorf("parse price_ref: %w", err)
	}

	signaturesRef, err := cellFromBase64(pr.ResultMessage.SignaturesRef)
	if err != nil {
		return nil, fmt.Errorf("parse signatures_ref: %w", err)
	}

	return &SignedPrice{PriceRef: priceRef, SignaturesRef: signaturesRef}, nil
}

func BuildSimplePayload(price *SignedPrice) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(0, 8).
		MustStoreRef(cell.BeginCell().
			MustStoreRef(price.PriceRef).
			MustStoreRef(price.SignaturesRef).
			EndCell()).
		MustStoreMaybeRef(nil).
		EndCell()
}

func BuildSettlementPayload(base, settlement *SignedPrice) *cell.Cell {
	return cell.BeginCell().
		MustStoreUInt(1, 8).
		MustStoreRef(cell.BeginCell().
			MustStoreRef(base.PriceRef).
			MustStoreRef(base.SignaturesRef).
			MustStoreRef(settlement.PriceRef).
			MustStoreRef(settlement.SignaturesRef).
			EndCell()).
		MustStoreMaybeRef(nil).
		EndCell()
}

func cellFromBase64(s string) (*cell.Cell, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return cell.FromBOC(data)
}
