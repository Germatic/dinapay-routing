package core

import "time"

type RouteRequest struct {
	RequestID           string   `json:"requestId"`
	TransactionID       string   `json:"transactionId"`
	AccountID           string   `json:"accountId,omitempty"`
	MerchantID          string   `json:"merchantId"`
	Operation           string   `json:"operation"`
	Amount              string   `json:"amount"`
	Currency            string   `json:"currency"`
	MarketCountry       string   `json:"marketCountry"`
	PaymentMethod       string   `json:"paymentMethod"`
	CustomerHasDocument bool     `json:"customerHasDocument,omitempty"`
	Rail                string   `json:"rail,omitempty"`
	DestinationMode     string   `json:"destinationMode,omitempty"`
	RequiredFeatures    []string `json:"requiredFeatures,omitempty"`
}
type BindingRequirement struct {
	EntityType         string `json:"entityType"`
	ExternalEntityType string `json:"externalEntityType"`
}
type Registration struct {
	MerchantID           string              `json:"merchantId,omitempty"`
	ConnectorID          string              `json:"connectorId"`
	Provider             string              `json:"provider"`
	ProviderConnectionID string              `json:"providerConnectionId"`
	Countries            []string            `json:"countries"`
	Currencies           []string            `json:"currencies"`
	PaymentMethods       []string            `json:"paymentMethods"`
	Rails                []string            `json:"rails"`
	DestinationModes     []string            `json:"destinationModes,omitempty"`
	Features             []string            `json:"features,omitempty"`
	BindingRequirement   *BindingRequirement `json:"bindingRequirement,omitempty"`
	Active               bool                `json:"active"`
}

type RoutingSnapshot struct {
	Version string         `json:"version"`
	Routes  []RuntimeRoute `json:"routes"`
}

type RuntimeRoute struct {
	MerchantID           string               `json:"merchantId"`
	Environment          string               `json:"environment"`
	APIVersion           string               `json:"apiVersion"`
	ConnectorID          string               `json:"connectorId"`
	Provider             string               `json:"provider"`
	ProviderConnectionID string               `json:"providerConnectionId"`
	Operation            string               `json:"operation"`
	Countries            []string             `json:"countries"`
	Currencies           []string             `json:"currencies"`
	PaymentMethods       []string             `json:"paymentMethods"`
	Rails                []string             `json:"rails"`
	DestinationModes     []string             `json:"destinationModes,omitempty"`
	Features             []string             `json:"features,omitempty"`
	BindingRequirements  []BindingRequirement `json:"bindingRequirements,omitempty"`
}
type ProviderBinding struct {
	BindingID          string `json:"bindingId"`
	EntityType         string `json:"entityType"`
	EntityID           string `json:"entityId"`
	ExternalEntityType string `json:"externalEntityType"`
	ExternalEntityID   string `json:"externalEntityId"`
}
type RouteDecision struct {
	RouteDecisionID      string           `json:"routeDecisionId"`
	RequestID            string           `json:"requestId"`
	TransactionID        string           `json:"transactionId"`
	Status               string           `json:"status"`
	ConnectorID          string           `json:"connectorId"`
	Provider             string           `json:"provider"`
	ProviderConnectionID string           `json:"providerConnectionId"`
	Rail                 string           `json:"rail"`
	DestinationMode      string           `json:"destinationMode,omitempty"`
	Binding              *ProviderBinding `json:"binding,omitempty"`
	PolicyVersion        string           `json:"policyVersion"`
	DecidedAt            time.Time        `json:"decidedAt"`
	ReasonCodes          []string         `json:"reasonCodes"`
}
type Rejection struct {
	ConnectorID          string `json:"connectorId,omitempty"`
	ProviderConnectionID string `json:"providerConnectionId,omitempty"`
	Code                 string `json:"code"`
	Detail               string `json:"detail,omitempty"`
}
type NoRoute struct {
	RouteDecisionID string      `json:"routeDecisionId"`
	RequestID       string      `json:"requestId"`
	TransactionID   string      `json:"transactionId"`
	Status          string      `json:"status"`
	PolicyVersion   string      `json:"policyVersion"`
	DecidedAt       time.Time   `json:"decidedAt"`
	Rejections      []Rejection `json:"rejections"`
}
type ZenDecision struct {
	Provider             string
	ProviderConnectionID string
	Rail                 string
	PaymentMethod        string
	Decision             string
	RuleID               string
	Reason               string
}
