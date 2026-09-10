# Dinapay Routing

Provider-neutral V2 routing service. ZEN remains authoritative and fail-closed;
this service adapts the V2 route contract to the existing ZEN Agent, validates
the decision against connector capabilities, resolves merchant bindings and
stores an auditable idempotent decision.

Target contract: `Germatic/dinapay-contracts` commit `c75b678`.

## Configuration

- `DB_URL`: shared PostgreSQL database.
- `ZEN_URL`: ZEN Agent URL, for example `http://127.0.0.1:8101`.
- `ZEN_PROJECT`: defaults to `default`.
- `ZEN_DECISION`: defaults to `payin_routing`.
- `ZEN_PAYOUT_DECISION`: defaults to `payout_routing`.
- `ZEN_POLICY_VERSION`: immutable rule deployment identifier.
- `SERVICE_TOKEN`: bearer token accepted from Dinapay V2.
- `CONTROL_PLANE_URL` and `CONTROL_PLANE_TOKEN`: when set, the router loads a
  merchant-scoped operational snapshot at startup and refreshes it every five
  seconds. Payment requests read only the in-memory last-known-good snapshot.
- `ENVIRONMENT` and `API_VERSION`: select the projected snapshot (defaults:
  `sandbox` and `v2`).
- `ROUTES_JSON`: legacy/static fallback when `CONTROL_PLANE_URL` is unset; see
  `config/routes.example.json`.

Network addresses and credentials are not routing data. `connectorId` is later
resolved by Dinapay V2 through service discovery.

ZEN remains the policy evaluator. The control plane snapshot defines which
connector capabilities a merchant is operationally entitled to use; it is not
copied into ZEN. An unavailable refresh never empties a working router: the
last valid snapshot remains active, while startup fails closed if no initial
snapshot can be obtained.
