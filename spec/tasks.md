# Implementation Tasks

**Deadline:** 1 Oct 2026
**Budget:** ~8.25 h to submittable (end of Phase 7), plus 1 h docs, plus 1.5–2 h bonus. Spec suggests 5–10 h. Adjust per-phase estimates once the deadline is set.
**Cut line:** if Phase 7 is not green by ~6 h spent, skip Phase 8 (bonus), document it in the README, and go to Phase 9.

Legend: `[ ]` todo · **dep:** must be done first · ⏱ estimate

---

## Phase 0 — Baseline, tooling & schema (⏱ 1 h)

- [ ] **T0.1** Environment (design §7.1): Git Bash on Windows with `GOTOOLCHAIN=go1.21.13` for the dev loop (the deadline is tomorrow, so no WSL migration now). Install `oapi-codegen` **v1.16.3** and `mockgen` (`go.uber.org/mock`) **v0.4.0**. WSL is used only for the final dry run (T9.4).
- [ ] **T0.2** `.gitattributes` (LF), then commit the untouched starter as the baseline commit. **dep:** T0.1
- [ ] **T0.3** Run `./run.sh` on the untouched starter and note what fails. The expected failure is the DB init. Don't fix anything yet. **dep:** T0.2
- [ ] **T0.4** `database.sql`: the real `estate` and `tree` schema per design §3 (CHECKs, FK, `UNIQUE (estate_id, x, y) INCLUDE (height)`). Verify with `docker compose down --volumes && docker compose up db`, then in psql: the tables exist, and a duplicate plot insert raises `23505`. *(Merged from old Phase 2.)*
- [ ] **T0.5** Compose: healthcheck `pg_isready -h localhost -U postgres -d database` with `start_period`; app `restart: on-failure`. Makefile: `test_unit` alias, coverage report that excludes `generated/` and `*.mock.gen.go`. Switch `golang/mock` → `go.uber.org/mock` v0.4.0 (`go.mod`, regenerate mock). Confirm `go.mod` stays `go 1.21`.

## Phase 1 — API contract (⏱ 1 h) · dep: Phase 0

- [ ] **T1.1** Write `api.yml`: 4 paths, `operationId`s, `ErrorResponse`, request and response schemas with `minimum`/`maximum`/`required`/`format: uuid`/`format: int64`, and optional `max_distance` query param and `rest` object (design §4). Remove `/hello` + `HelloResponse` and the stale Gradle header comment; add `info.description`.
- [ ] **T1.2** `make generate`. Remove the `Hello` handler, its unit test, and the `/hello` cases in `tests/api_test.go` (done here, not in Phase 0, because removing `/hello` from the spec breaks the build until the stubs exist). Stub every `ServerInterface` method so the project compiles. Read the generated param/body types and note any surprises (pointer vs value for optional fields).

## Phase 2 — Schema · merged into T0.4

## Phase 3 — Pure domain (⏱ 2 h) · dep: none (can run in parallel with Phases 1–2)

- [ ] **T3.1** `droneplan`: path index ↔ (x, y) conversion, with tests including a single row, a single column, and even/odd rows.
- [ ] **T3.2** `droneplan`: total distance, O(T log T), `int64`. Table tests from design §5.2 (54, 82, 1×1, empty, row-turn adjacency, 50000² with 3 trees).
- [ ] **T3.3** Test-only brute-force O(N) simulator plus a randomized comparison test (small L, W, random trees, fixed seed). **dep:** T3.2
- [ ] **T3.4** `treestats`: histogram → count/min/max/median, with table tests (empty, 1, odd, even → x.5, all-same).
- ✅ Checkpoint: `go test ./droneplan/... ./treestats/...` shows ~100% coverage.

## Phase 4 — Repository (⏱ 1.25 h) · dep: Phase 0

- [ ] **T4.1** `interfaces.go`: `CreateEstate`, `GetEstate`, `CreateTree`, `ListTrees` (x, y, height), `GetHeightHistogram`. Domain errors `ErrNotFound`, `ErrPlotOccupied`. Regenerate the mocks.
- [ ] **T4.2** Postgres implementation: context-aware queries, `RETURNING id`, error translation (`ErrNoRows`, `23505`, `23503`).
- [ ] **T4.3** `go-sqlmock` tests for the translation paths only (don't mock-test happy-path SQL text).

## Phase 5 — Handlers (⏱ 1.5 h) · dep: Phases 1, 3, 4

- [ ] **T5.1** Custom `HTTPErrorHandler`: `{message}` always, 500s generic plus logged with the request ID.
- [ ] **T5.2** `POST /estate` and `POST /estate/{id}/tree`: 201; bounds check against the fetched estate; error mapping.
- [ ] **T5.3** `GET /stats` and `GET /drone-plan` (without the bonus). Leave the `max_distance` branch returning just `distance` for now.
- [ ] **T5.4** gomock handler tests: one per status branch per endpoint.

## Phase 6 — Wiring & operability (⏱ 0.75 h) · dep: Phase 5

- [ ] **T6.1** `main.go`: env config, DB ping with bounded retry, pool limits, `http.Server` timeouts, graceful shutdown.
- [ ] **T6.2** Middleware chain: Recover → RequestID → Logger → BodyLimit → OpenAPI request validator (clear `spec.Servers`). Verify that empty body, `10.5`, `"10"`, and a malformed UUID all return 400 with a `{message}` JSON body.

## Phase 7 — End-to-end (⏱ 0.75 h) · dep: Phase 6

- [ ] **T7.1** Update `tests/api_test.go`: 200 → 201 for creates; add 404s, occupied plot, out of bounds, even median, malformed UUID, empty body.
- [ ] **T7.2** Full `run.sh` from `docker compose down --volumes`: everything green, coverage ≥ 70%.
- ✅ Checkpoint: **submittable.** Everything after this is upside.

## Phase 8 — BONUS: `max_distance` → `rest` (⏱ 1.5–2 h, hard time-box) · dep: Phase 7

- [ ] **T8.1** `droneplan`: forced-landing point per design §5.3 (segment attribution, exact-boundary rule, arithmetic skip over empty runs, `≥ total` → last plot).
- [ ] **T8.2** Extend the brute-force simulator to produce the landing plot. Run the randomized comparison over every `max_distance` in `1..total+5` for small estates. This is the test that lets you claim "correct".
- [ ] **T8.3** Handler branch plus unit tests; API tests for rest mid-climb, mid-horizontal, mid-descent, exact boundary, and above the total.

## Phase 9 — Docs & submission (⏱ 1 h) · dep: Phase 7 (and 8 if done)

- [ ] **T9.1** README: 30-second quickstart (`./run.sh`), curl examples for all endpoints, project layout, testing commands, **Assumptions** table (copy of design §8), and not-built list.
- [ ] **T9.2** Trim `spec/design.md` for the zip. Keep `spec/tasks.md` out of the zip.
- [ ] **T9.3** `gofmt`/`go vet` clean. Check that comments explain *why* rather than *what*. Remove dead code.
- [ ] **T9.4** Fresh-copy dry run: unzip into a new directory under WSL and run `./run.sh` end to end. Then zip and submit.

---

## Time summary

| Phase | ⏱ | Cumulative |
|---|---|---|
| 0 Baseline, tooling & schema | 1.0 | 1.0 |
| 1 Contract | 1.0 | 2.0 |
| 2 Schema (merged into 0) | – | 2.0 |
| 3 Pure domain | 2.0 | 4.0 |
| 4 Repository | 1.25 | 5.25 |
| 5 Handlers | 1.5 | 6.75 |
| 6 Wiring | 0.75 | 7.5 |
| 7 E2E ✅ submittable | 0.75 | 8.25 |
| 8 Bonus | 1.5–2 | ~10 |
| 9 Docs & submit | 1.0 | ~11 |

About 11 h all-in is above the spec's 5–10 h range. Protect the time by either compressing Phases 3–5 (you've done this pattern many times) or dropping Phase 8 at the cut line. **Do Phase 9 even if you skip Phase 8.** A clear README scores more than an unfinished bonus.
