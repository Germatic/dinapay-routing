package controlplane

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRefreshPublishesValidatedRuntimeRoutes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer runtime-secret" {
			t.Fatalf("authorization=%q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"version":"v1","routes":[{"merchantId":"merchant1","environment":"sandbox","apiVersion":"v2","connectorId":"connector-binancepay-v2","provider":"binancepay","providerConnectionId":"main","operation":"payment","countries":["*"],"currencies":["USDT"],"paymentMethods":["crypto_payment"],"rails":["binance_pay"],"features":["refund"],"bindingRequirements":[{"entityType":"merchant","externalEntityType":"sub_merchant"}]}]}`))
	}))
	defer server.Close()
	source := New(server.URL, "runtime-secret", "sandbox", "v2")
	if err := source.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	routes := source.Current()
	if len(routes) != 1 || routes[0].MerchantID != "merchant1" || routes[0].BindingRequirement == nil || routes[0].BindingRequirement.ExternalEntityType != "sub_merchant" {
		t.Fatalf("routes=%#v", routes)
	}
}

func TestRefreshKeepsLastKnownGoodSnapshotOnError(t *testing.T) {
	fail := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			http.Error(w, "down", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(`{"version":"v1","routes":[{"merchantId":"merchant1","connectorId":"connector","provider":"provider","providerConnectionId":"main","operation":"payment","countries":["*"],"currencies":["USD"],"paymentMethods":["bank_transfer"],"rails":["ach"]}]}`))
	}))
	defer server.Close()
	source := New(server.URL, "token", "sandbox", "v2")
	if err := source.Refresh(context.Background()); err != nil {
		t.Fatal(err)
	}
	fail = true
	if err := source.Refresh(context.Background()); err == nil {
		t.Fatal("expected refresh error")
	}
	if len(source.Current()) != 1 {
		t.Fatalf("last known good snapshot was discarded")
	}
}
