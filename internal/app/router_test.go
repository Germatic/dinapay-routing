package app

import (
	"context"
	"testing"

	"github.com/Germatic/dinapay-routing/internal/core"
)

type rulesStub struct {
	calls    int
	decision core.ZenDecision
}

func (r *rulesStub) Evaluate(_ context.Context, _ core.RouteRequest) (core.ZenDecision, error) {
	r.calls++
	return r.decision, nil
}

type savedDecision struct {
	hash string
	body []byte
}
type decisionStub struct {
	saved   map[string]savedDecision
	binding *core.ProviderBinding
}

func (d *decisionStub) Find(_ context.Context, id string) (string, []byte, bool, error) {
	v, ok := d.saved[id]
	return v.hash, v.body, ok, nil
}
func (d *decisionStub) Save(_ context.Context, id, hash string, body []byte) ([]byte, bool, error) {
	if v, ok := d.saved[id]; ok {
		if v.hash != hash {
			return nil, false, core.ErrConflict
		}
		return v.body, true, nil
	}
	d.saved[id] = savedDecision{hash, body}
	return body, false, nil
}
func (d *decisionStub) Binding(_ context.Context, _, _, _, _, _ string) (*core.ProviderBinding, error) {
	return d.binding, nil
}

func TestResolveUsesZenCapabilitiesBindingAndIdempotency(t *testing.T) {
	rules := &rulesStub{decision: core.ZenDecision{Provider: "binancepay", ProviderConnectionID: "binancepay_sandbox_main", Rail: "binance_pay", Decision: "route", RuleID: "payin.usdt.binancepay"}}
	store := &decisionStub{saved: map[string]savedDecision{}, binding: &core.ProviderBinding{BindingID: "binding1", EntityType: "merchant", EntityID: "merchant1", ExternalEntityType: "sub_merchant", ExternalEntityID: "123"}}
	registrations := []core.Registration{{ConnectorID: "connector-binancepay-v2", Provider: "binancepay", ProviderConnectionID: "binancepay_sandbox_main", Countries: []string{"*"}, Currencies: []string{"USDT"}, PaymentMethods: []string{"crypto_payment"}, Rails: []string{"binance_pay"}, Features: []string{"refund"}, BindingRequirement: &core.BindingRequirement{EntityType: "merchant", ExternalEntityType: "sub_merchant"}, Active: true}}
	router := New(rules, store, registrations, "rules-1")
	in := core.RouteRequest{RequestID: "11111111-1111-4111-8111-111111111111", TransactionID: "22222222-2222-4222-8222-222222222222", AccountID: "account1", MerchantID: "merchant1", Operation: "payment", Amount: "0.25", Currency: "USDT", MarketCountry: "UY", PaymentMethod: "crypto_payment", RequiredFeatures: []string{"refund"}}
	first, err := router.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	second, err := router.Resolve(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if first.RouteDecisionID != second.RouteDecisionID || rules.calls != 1 || first.Rail != "binance_pay" || first.Binding == nil || first.Binding.ExternalEntityID != "123" {
		t.Fatalf("first=%#v second=%#v calls=%d", first, second, rules.calls)
	}
}

func TestResolveInfersSingleDestinationMode(t *testing.T) {
	rules := &rulesStub{decision: core.ZenDecision{Provider: "transferdirecto", ProviderConnectionID: "transferdirecto_main", Rail: "spei", Decision: "route", RuleID: "payin.mxn.default"}}
	store := &decisionStub{saved: map[string]savedDecision{}, binding: &core.ProviderBinding{BindingID: "binding1", EntityType: "merchant", EntityID: "merchant1", ExternalEntityType: "cost_center", ExternalEntityID: "37"}}
	registrations := []core.Registration{{ConnectorID: "connector-transferdirecto-v2", Provider: "transferdirecto", ProviderConnectionID: "transferdirecto_main", Countries: []string{"MX"}, Currencies: []string{"MXN"}, PaymentMethods: []string{"bank_transfer"}, Rails: []string{"spei"}, DestinationModes: []string{"single_use"}, BindingRequirement: &core.BindingRequirement{EntityType: "merchant", ExternalEntityType: "cost_center"}, Active: true}}
	router := New(rules, store, registrations, "rules-1")
	out, err := router.Resolve(context.Background(), core.RouteRequest{RequestID: "11111111-1111-4111-8111-111111111111", TransactionID: "22222222-2222-4222-8222-222222222222", AccountID: "account1", MerchantID: "merchant1", Operation: "payment", Amount: "100.00", Currency: "MXN", MarketCountry: "MX", PaymentMethod: "bank_transfer"})
	if err != nil || out.Rail != "spei" || out.DestinationMode != "single_use" {
		t.Fatalf("out=%#v err=%v", out, err)
	}
}

func TestResolveRejectsMissingBinding(t *testing.T) {
	rules := &rulesStub{decision: core.ZenDecision{Provider: "binancepay", Decision: "route", RuleID: "rule"}}
	store := &decisionStub{saved: map[string]savedDecision{}}
	router := New(rules, store, []core.Registration{{ConnectorID: "binance", Provider: "binancepay", ProviderConnectionID: "main", Countries: []string{"*"}, Currencies: []string{"USDT"}, PaymentMethods: []string{"crypto_payment"}, BindingRequirement: &core.BindingRequirement{EntityType: "merchant", ExternalEntityType: "sub_merchant"}, Active: true}}, "rules-1")
	_, err := router.Resolve(context.Background(), core.RouteRequest{RequestID: "1", TransactionID: "2", MerchantID: "merchant1", Operation: "payment", Amount: "1.00", Currency: "USDT", MarketCountry: "UY", PaymentMethod: "crypto_payment"})
	no, ok := err.(NoRouteError)
	if !ok || no.Decision.Rejections[0].Code != "binding_missing" {
		t.Fatalf("unexpected error: %#v", err)
	}
}
