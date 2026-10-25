# EV_PNC_telemetry

Charger data foundation: live status ingest with a 15-minute freshness rule, status/fault/occupancy history, EPDK list reconciliation, corridor EVSE index. Go services; telemetry-api for consumers.

Part of the **EV_PNC** platform: a neutral eMSP integration layer for Turkey-first, Europe-ready Plug&Charge, navigation and payments. The platform is neither a CPO nor a payment institution.

> Plug in, charging starts, payment completes in the background.

## Scope

- Sources: bilateral OCPI, hubs, EPDK station list (weekly reconcile, CP-DATA-004), later the national data access point
- Freshness rule and provenance per field (CP-DATA-001)
- History and profiles for queue prediction (CP-DATA-002/003)
- Corridor geometries M, A1 to A4 (versioned) and the EVSE index with family and Oksijen cluster tags (CP-GEO-001/002)
- Availability cache for the planner and the app

## Tech stack

Go, Redis, Kafka, TimescaleDB

## Packaging

- Published components: telemetry-api, telemetry-server
- Depends on: EV_PNC_ocpi-model
- Depended on by: EV_PNC_api-gateway, EV_PNC_pilot-analytics, EV_PNC_planner
- Dependency rules: no cycles (ADP); depend towards more stable components (SDP); published `*-api` components hold abstractions, deployables hold concrete code (SAP). CI runs `tools/packaging_check.py --strict` from `EV_PNC_docs`.
- Owner: Backend agent · Wave 1 (now, before the 2026-12-07 gate) · Visibility: public

## Proposed layout

```
cmd/{ingest,history,api}
internal/sources/{ocpi,hub,epdk}
corridor/geometries/
```

## Related

- Plan: `repo-plan/v2/README.md` (project files); consent repos: `EV_PnC_Consent_*` (`repo-plan/consent/README.md`)
- Docs: [EV_PNC_docs](https://github.com/COPUR/EV_PNC_docs)

## Status

Scaffold only. Not production ready.

## License

Apache-2.0 (proposed)
