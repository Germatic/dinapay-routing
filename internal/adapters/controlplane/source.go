package controlplane

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/Germatic/dinapay-routing/internal/core"
)

type Source struct {
	url, token, environment, apiVersion string
	http                                *http.Client
	routes                              atomic.Value
}

func New(baseURL, token, environment, apiVersion string) *Source {
	s := &Source{url: baseURL, token: token, environment: environment, apiVersion: apiVersion, http: &http.Client{Timeout: 3 * time.Second}}
	s.routes.Store([]core.Registration{})
	return s
}

func (s *Source) Current() []core.Registration { return s.routes.Load().([]core.Registration) }

func (s *Source) Refresh(ctx context.Context) error {
	endpoint := s.url + "/internal/v1/routing-snapshot?environment=" + url.QueryEscape(s.environment) + "&apiVersion=" + url.QueryEscape(s.apiVersion)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+s.token)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("control plane returned %s", resp.Status)
	}
	var snapshot core.RoutingSnapshot
	if err = json.NewDecoder(resp.Body).Decode(&snapshot); err != nil {
		return err
	}
	routes := make([]core.Registration, 0, len(snapshot.Routes))
	for _, r := range snapshot.Routes {
		if r.Operation != "payment" && r.Operation != "payout" {
			continue
		}
		if r.MerchantID == "" || r.ConnectorID == "" || r.Provider == "" || r.ProviderConnectionID == "" || len(r.Countries) == 0 || len(r.Rails) == 0 || (r.Operation == "payment" && (len(r.Currencies) == 0 || len(r.PaymentMethods) == 0)) || (r.Operation == "payout" && (len(r.SourceCurrencies) == 0 || len(r.DestinationCurrencies) == 0)) {
			return fmt.Errorf("control plane returned an invalid route")
		}
		if len(r.BindingRequirements) > 1 {
			return fmt.Errorf("multiple binding requirements are not supported by routing contract v1")
		}
		var requirement *core.BindingRequirement
		for _, candidate := range r.BindingRequirements {
			requirement = &core.BindingRequirement{EntityType: candidate.EntityType, ExternalEntityType: candidate.ExternalEntityType}
			break
		}
		routes = append(routes, core.Registration{Operation: r.Operation, MerchantID: r.MerchantID, ConnectorID: r.ConnectorID, Provider: r.Provider,
			ProviderConnectionID: r.ProviderConnectionID, Countries: r.Countries, Currencies: r.Currencies,
			SourceCurrencies: r.SourceCurrencies, DestinationCurrencies: r.DestinationCurrencies,
			PaymentMethods: r.PaymentMethods, Rails: r.Rails, DestinationModes: r.DestinationModes, Features: r.Features,
			BindingRequirement: requirement, Active: true})
	}
	s.routes.Store(routes)
	return nil
}

func (s *Source) Run(ctx context.Context, interval time.Duration, onError func(error)) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.Refresh(ctx); err != nil && onError != nil {
				onError(err)
			}
		}
	}
}
