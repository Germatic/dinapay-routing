package zen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/Germatic/dinapay-routing/internal/core"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL, project, decision, payoutDecision, token string
	http                                              *http.Client
}

func New(baseURL, project, decision, token string, payoutDecisions ...string) *Client {
	if project == "" {
		project = "default"
	}
	if decision == "" {
		decision = "payin_routing"
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{Timeout: time.Second, KeepAlive: 30 * time.Second}).DialContext
	transport.MaxIdleConns = 64
	transport.MaxIdleConnsPerHost = 32
	transport.MaxConnsPerHost = 64
	transport.IdleConnTimeout = 90 * time.Second
	transport.ResponseHeaderTimeout = 1500 * time.Millisecond
	payoutDecision := "payout_routing"
	if len(payoutDecisions) > 0 && payoutDecisions[0] != "" {
		payoutDecision = payoutDecisions[0]
	}
	return &Client{strings.TrimRight(baseURL, "/"), project, decision, payoutDecision, token, &http.Client{Timeout: 2 * time.Second, Transport: transport}}
}
func (c *Client) Evaluate(ctx context.Context, in core.RouteRequest) (core.ZenDecision, error) {
	amount := json.Number(in.Amount)
	flow := "payin"
	if in.Operation == "payout" {
		flow = "payout"
	}
	facts := map[string]any{"account_id": in.AccountID, "merchant_id": in.MerchantID, "currency": in.Currency, "destination_currency": in.DestinationCurrency, "country": in.MarketCountry, "flow": flow, "amount": amount, "has_document": in.CustomerHasDocument, "payment_method": in.PaymentMethod, "destination_identifier_type": in.Rail}
	body, err := json.Marshal(map[string]any{"context": facts})
	if err != nil {
		return core.ZenDecision{}, err
	}
	key := c.decision
	if in.Operation == "payout" {
		key = c.payoutDecision
	}
	if !strings.HasSuffix(key, ".json") {
		key += ".json"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/projects/%s/evaluate/%s", c.baseURL, c.project, key), bytes.NewReader(body))
	if err != nil {
		return core.ZenDecision{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	if requestID := core.RequestID(ctx); requestID != "" {
		req.Header.Set("X-Request-Id", requestID)
	}
	if traceparent := core.Traceparent(ctx); traceparent != "" {
		req.Header.Set("traceparent", traceparent)
	}
	if c.token != "" {
		req.Header.Set("X-Access-Token", c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return core.ZenDecision{}, err
	}
	defer resp.Body.Close()
	defer func() { _, _ = io.Copy(io.Discard, resp.Body) }()
	if resp.StatusCode >= 300 {
		return core.ZenDecision{}, fmt.Errorf("ZEN returned %s", resp.Status)
	}
	var raw struct {
		Result struct {
			Provider             string `json:"provider_code"`
			ProviderConnectionID string `json:"provider_account"`
			Rail                 string `json:"rail_code"`
			PaymentMethod        string `json:"payment_method_code"`
			Decision             string `json:"decision"`
			RuleID               string `json:"rule_id"`
			Reason               string `json:"reason"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return core.ZenDecision{}, err
	}
	return core.ZenDecision(raw.Result), nil
}
