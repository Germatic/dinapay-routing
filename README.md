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
- `ZEN_POLICY_VERSION`: immutable rule deployment identifier.
- `SERVICE_TOKEN`: bearer token accepted from Dinapay V2.
- `ROUTES_JSON`: connector registrations; see `config/routes.example.json`.

Network addresses and credentials are not routing data. `connectorId` is later
resolved by Dinapay V2 through service discovery.
