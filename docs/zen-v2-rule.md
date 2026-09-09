# ZEN rule requirements for V2

The router evaluates the existing `payin_routing.json` decision and remains
fail-closed. To activate a V2 provider, the rule output must include:

- `provider_code`
- `provider_account` (the V2 `providerConnectionId`)
- `rail_code`
- `payment_method_code`
- `decision=route`
- stable `rule_id`

For the first Binance Pay test, the sandbox table needs a rule matching
`payment_method=crypto_payment` and the intended currencies, returning
`provider_code=binancepay`, `provider_account=binancepay_sandbox_main` and
`rail_code=binance_pay`.

Rules are still deployed through the existing validated ZEN deployment flow.
The immutable deployment identifier must be passed as `ZEN_POLICY_VERSION` so
stored decisions can be reproduced during support and reconciliation.
