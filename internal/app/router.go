package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/Germatic/dinapay-routing/internal/core"
)

var decimalPattern = regexp.MustCompile(`^(0\.[0-9]*[1-9][0-9]*|[1-9][0-9]*(\.[0-9]+)?)$`)

var (
	ErrInvalid     = errors.New("invalid request")
	ErrConflict    = core.ErrConflict
	ErrUnavailable = errors.New("routing unavailable")
)

type NoRouteError struct{ Decision core.NoRoute }

func (e NoRouteError) Error() string { return "no eligible route" }

type Router struct {
	rules         core.Rules
	store         core.Decisions
	registrations []core.Registration
	policyVersion string
	now           func() time.Time
}

func New(rules core.Rules, store core.Decisions, registrations []core.Registration, policyVersion string) *Router {
	return &Router{rules: rules, store: store, registrations: registrations, policyVersion: policyVersion, now: time.Now}
}

func (s *Router) Resolve(ctx context.Context, in core.RouteRequest) (core.RouteDecision, error) {
	if in.RequestID == "" || in.TransactionID == "" || in.MerchantID == "" || in.Operation != "payment" || !decimalPattern.MatchString(in.Amount) || in.Currency == "" || in.MarketCountry == "" || in.PaymentMethod == "" || s.policyVersion == "" {
		return core.RouteDecision{}, ErrInvalid
	}
	hash := requestHash(in)
	if storedHash, body, ok, err := s.store.Find(ctx, in.RequestID); err != nil {
		return core.RouteDecision{}, err
	} else if ok {
		if storedHash != hash {
			return core.RouteDecision{}, ErrConflict
		}
		var out core.RouteDecision
		if json.Unmarshal(body, &out) == nil && out.Status == "selected" {
			return out, nil
		}
		var no core.NoRoute
		_ = json.Unmarshal(body, &no)
		return core.RouteDecision{}, NoRouteError{no}
	}
	zen, err := s.rules.Evaluate(ctx, in)
	if err != nil {
		return core.RouteDecision{}, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	now := s.now().UTC()
	decisionID := stableID("route:" + in.RequestID)
	if zen.Decision != "route" || zen.Provider == "" {
		no := core.NoRoute{RouteDecisionID: decisionID, RequestID: in.RequestID, TransactionID: in.TransactionID, Status: "no_route", PolicyVersion: s.policyVersion, DecidedAt: now, Rejections: []core.Rejection{{Code: "policy_rejected", Detail: zen.Reason}}}
		return core.RouteDecision{}, s.saveNoRoute(ctx, in.RequestID, hash, no)
	}
	registration, rejections, ok := selectRegistration(s.registrations, in, zen)
	if !ok {
		no := core.NoRoute{RouteDecisionID: decisionID, RequestID: in.RequestID, TransactionID: in.TransactionID, Status: "no_route", PolicyVersion: s.policyVersion, DecidedAt: now, Rejections: rejections}
		return core.RouteDecision{}, s.saveNoRoute(ctx, in.RequestID, hash, no)
	}
	var binding *core.ProviderBinding
	if requirement := registration.BindingRequirement; requirement != nil {
		entityID := in.MerchantID
		if requirement.EntityType == "account" {
			entityID = in.AccountID
		}
		binding, err = s.store.Binding(ctx, requirement.EntityType, entityID, registration.Provider, registration.ProviderConnectionID, requirement.ExternalEntityType)
		if err != nil {
			return core.RouteDecision{}, err
		}
		if binding == nil {
			no := core.NoRoute{RouteDecisionID: decisionID, RequestID: in.RequestID, TransactionID: in.TransactionID, Status: "no_route", PolicyVersion: s.policyVersion, DecidedAt: now, Rejections: []core.Rejection{{ConnectorID: registration.ConnectorID, ProviderConnectionID: registration.ProviderConnectionID, Code: "binding_missing"}}}
			return core.RouteDecision{}, s.saveNoRoute(ctx, in.RequestID, hash, no)
		}
	}
	reasonCode := zen.RuleID
	if reasonCode == "" {
		reasonCode = "zen.route"
	}
	out := core.RouteDecision{RouteDecisionID: decisionID, RequestID: in.RequestID, TransactionID: in.TransactionID, Status: "selected", ConnectorID: registration.ConnectorID, Provider: registration.Provider, ProviderConnectionID: registration.ProviderConnectionID, Binding: binding, PolicyVersion: s.policyVersion, DecidedAt: now, ReasonCodes: []string{reasonCode}}
	body, _ := json.Marshal(out)
	saved, _, err := s.store.Save(ctx, in.RequestID, hash, body)
	if err != nil {
		return core.RouteDecision{}, err
	}
	_ = json.Unmarshal(saved, &out)
	return out, nil
}
func (s *Router) saveNoRoute(ctx context.Context, id, hash string, no core.NoRoute) error {
	body, _ := json.Marshal(no)
	saved, _, err := s.store.Save(ctx, id, hash, body)
	if err != nil {
		return err
	}
	_ = json.Unmarshal(saved, &no)
	return NoRouteError{no}
}
func requestHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func stableID(value string) string {
	h := sha256.Sum256([]byte(value))
	b := h[:16]
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
func selectRegistration(all []core.Registration, in core.RouteRequest, zen core.ZenDecision) (core.Registration, []core.Rejection, bool) {
	var eligible []core.Registration
	var rejected []core.Rejection
	for _, r := range all {
		if !r.Active || r.Provider != zen.Provider {
			continue
		}
		code := ""
		if zen.ProviderConnectionID != "" && r.ProviderConnectionID != zen.ProviderConnectionID {
			code = "connection_disabled"
		} else if !contains(r.Countries, in.MarketCountry) {
			code = "country_unsupported"
		} else if !contains(r.Currencies, in.Currency) {
			code = "currency_unsupported"
		} else if !contains(r.PaymentMethods, in.PaymentMethod) {
			code = "method_unsupported"
		} else if in.Rail != "" && !contains(r.Rails, in.Rail) {
			code = "rail_unsupported"
		} else if zen.Rail != "" && !contains(r.Rails, zen.Rail) {
			code = "rail_unsupported"
		} else if in.DestinationMode != "" && !contains(r.DestinationModes, in.DestinationMode) {
			code = "destination_mode_unsupported"
		} else {
			for _, f := range in.RequiredFeatures {
				if !contains(r.Features, f) {
					code = "feature_unsupported"
					break
				}
			}
		}
		if code != "" {
			rejected = append(rejected, core.Rejection{ConnectorID: r.ConnectorID, ProviderConnectionID: r.ProviderConnectionID, Code: code})
			continue
		}
		eligible = append(eligible, r)
	}
	if len(eligible) != 1 {
		if len(rejected) == 0 {
			rejected = []core.Rejection{{Code: "policy_rejected", Detail: "provider decision did not resolve exactly one active connector registration"}}
		}
		return core.Registration{}, rejected, false
	}
	return eligible[0], nil, true
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == "*" || v == want {
			return true
		}
	}
	return false
}
