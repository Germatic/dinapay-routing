package zen

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Germatic/dinapay-routing/internal/core"
)

func TestEvaluateUsesPayoutDecisionAndFacts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/default/evaluate/custom_payout.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		var body struct {
			Context map[string]any `json:"context"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Context["flow"] != "payout" || body.Context["destination_identifier_type"] != "ve_mobile_payment" || body.Context["destination_currency"] != "VES" {
			t.Fatalf("context=%#v", body.Context)
		}
		_, _ = w.Write([]byte(`{"result":{"provider_code":"insular","decision":"route"}}`))
	}))
	defer server.Close()

	client := New(server.URL, "default", "payin_routing", "", "custom_payout")
	got, err := client.Evaluate(t.Context(), core.RouteRequest{Operation: "payout", Amount: "1.00", Currency: "USDT", DestinationCurrency: "VES", Rail: "ve_mobile_payment"})
	if err != nil || got.Provider != "insular" {
		t.Fatalf("result=%#v err=%v", got, err)
	}
}

func TestEvaluateKeepsPayinDecision(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/projects/default/evaluate/payin_routing.json" {
			t.Fatalf("path=%s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"result":{"provider_code":"binancepay","decision":"route"}}`))
	}))
	defer server.Close()

	_, err := New(server.URL, "default", "payin_routing", "").Evaluate(t.Context(), core.RouteRequest{Operation: "payment", Amount: "1.00"})
	if err != nil {
		t.Fatal(err)
	}
}
