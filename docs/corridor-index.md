# Corridor model and EVSE index (cards C-01, T-CP-01, T-CP-02)

## What exists

- **Geometries** `corridor/geometries/<version>/`: one GeoJSON `LineString` per family (M, A1, A2, A3, A4) plus a `manifest.json` that pins each file by SHA-256. `versions.json` pins each manifest. A published version is immutable: a test fails if any file or manifest changes, and the fix is a new version directory.
- **Index builder** `cmd/corridor-index`: reads EVSE records, tags each one within the buffer (default 2 km, CP-GEO-002) of a family geometry with the family codes it serves and the distance to each, and assigns a cluster label from `corridor/clusters.json`. Output is deterministic (sorted, no timestamp).
- **K1 evidence flag**: records with `source: user_listed` are indexed if they have coordinates but carry `k1_evidence: false` and a reason. Everything else is `true`. Reachability per channel is a separate step (card C-03 reads this index).
- **Unplaced list**: records without coordinates, out of range, or outside every buffer are reported with the reason, never dropped silently.
- `corridor/candidates/user-listed.json`: the stations named in `corridor-pilot/scope.md`, marked K, no coordinates yet.

```bash
go run ./cmd/corridor-index --evse evse.json --out index.json
```

Input record fields: `evse_uid` (required, unique), `source` (`user_listed`, `epdk_list`, `ocpi_feed`, `hub_feed`), `lat`, `lon`, and optional `evse_id`, `location_id`, `site_id`, `name`, `operator`, `max_power_kw`, `power_type`.

## What is not done, and why

1. **The v0 geometries are draft.** They are hand-placed city-level waypoints (`quality: draft-waypoints`), good for wiring and tests. A 2 km buffer around them says nothing true about real roads. Before any in-corridor decision or K1 count, publish `v1` from a routing engine run over the real road network (needs a routing data source and its licence; decision for the product owner). The index output carries `geometry_quality` so downstream reports can refuse draft input.
2. **"Rebuilt nightly" needs a scheduler and an EVSE source.** The CLI is idempotent and ready to run as a cron job; the job itself waits for the first environment (EV_PNC_infra) and for the OCPI/hub/EPDK adapters (T-CP-10, T-CP-12), which produce the input records.
3. **Oksijen cluster membership is unknown (U).** `kpis.md` names the clusters but does not list them. The placeholder rule matches the station name; replace it with site ids from CPO data.
4. The user-listed stations still need coordinates and verification against the EPDK list and CPO offers (CP-DATA-004).
