# Estate Patrol Service

Record plantation estates and the trees in them, report tree height statistics, and plan drone patrols, including the battery-limited bonus.

## Quick start

```bash
./run.sh
```

This runs the full pipeline: generate code, build, unit tests with coverage, start Postgres and the API with Docker, run the API tests, and shut everything down. Look for `ok` on every test package and `total: … 99.5%` in the output. `run.sh` always exits 0 because its last command is `docker compose down`, so the output is what tells you whether it passed.

### Requirements

| Tool | Version | Install |
|---|---|---|
| Go | 1.21 | built and tested with 1.21.13 |
| GNU Make | any | |
| oapi-codegen | v1.16.3 | `go install github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.16.3` |
| mockgen | v0.4.0 | `go install go.uber.org/mock/mockgen@v0.4.0` (newer releases need Go > 1.21) |
| Docker | 20.10+ with Compose v2 (`docker compose`) | |

### Running pieces by hand

```bash
make init                 # generate code and mocks, tidy and vendor modules
make                      # build/main
make test                 # unit tests + coverage (alias: make test_unit)
docker compose up --build # API on http://localhost:8080
make test_api             # API tests against the running stack
docker compose down --volumes   # also needed after changing database.sql
```

## API

| Method and path | Success | Errors |
|---|---|---|
| `POST /estate` | `201 {"id"}` | 400 invalid body |
| `POST /estate/{id}/tree` | `201 {"id"}` | 400 invalid body, plot outside the estate, or plot already has a tree; 404 unknown estate |
| `GET /estate/{id}/stats` | `200 {"count","min","max","median"}` | 400 malformed id; 404 unknown estate |
| `GET /estate/{id}/drone-plan` | `200 {"distance"}` | 400 malformed id; 404 unknown estate |
| `GET /estate/{id}/drone-plan?max_distance=N` | `200 {"distance","rest":{"x","y"}}` | 400 if N is not a positive integer |

Every error is `{"message": "..."}`. The full contract is in [api.yml](api.yml).

```bash
curl -s -X POST localhost:8080/estate -H 'Content-Type: application/json' -d '{"length":5,"width":1}'
# {"id":"<estate-id>"}

curl -s -X POST localhost:8080/estate/<estate-id>/tree -H 'Content-Type: application/json' -d '{"x":2,"y":1,"height":5}'
# repeat with {"x":3,"y":1,"height":3} and {"x":4,"y":1,"height":4}

curl -s localhost:8080/estate/<estate-id>/stats
# {"count":3,"max":5,"median":4,"min":3}

curl -s localhost:8080/estate/<estate-id>/drone-plan
# {"distance":54}

curl -s 'localhost:8080/estate/<estate-id>/drone-plan?max_distance=27'
# {"distance":27,"rest":{"x":3,"y":1}}
```

## How it works

### Patrol distance

An estate can be 50,000 × 50,000 = 2.5 billion plots, so nothing loops over plots or allocates a grid. The drone flies a zig-zag (odd rows east, even rows west), so every step is 10 m, including the step north at a row end. That turns the estate into a single line of N plots:

```
distance = 10·(N−1) + 2 + sum of height changes along the path
```

The `2` is take-off and landing at 1 m. Trees are sorted by their position on the path. Two neighbouring trees cost `|h1 − h2|`; trees with empty plots between them cost `h1 + h2` (down to 1 m and back up), however long the gap. That makes it **O(T log T) in the number of trees**, independent of the estate size. All distances are `int64`.

### Bonus: forced landing

The drone climbs before moving onto a taller plot and descends after moving onto a lower one. If the main battery runs out:

| …during | it lands on |
|---|---|
| a climb at plot p | p |
| a move from p to p+1 | p, the last plot it reached |
| a descent after reaching p+1 | p+1 |
| nothing (the battery covers the whole plan) | the last plot |

With `max_distance`, `distance` is the distance flown before landing, `min(full plan, max_distance)`. The landing descent uses the backup battery and isn't counted. Runs of empty plots are skipped with one division, so this is O(T log T) as well.

### Stats

`SELECT height, count(*) … GROUP BY height` returns at most 30 rows (heights are 1–30) however many trees there are. Count, min, max and median are computed from that histogram. The median of an even count is the mean of the two middle heights, e.g. 3.5.

## Design decisions

- **One tree per plot is a database constraint**, `UNIQUE (estate_id, x, y)`, not a check-then-insert in Go, so concurrent requests can't both win. Checked: 50 concurrent inserts on one plot give exactly one 201. The same index covers the foreign key and both read queries (`INCLUDE (height)`).
- **"Plot inside the estate" is checked in the handler** against the estate it just read. That's safe without a transaction because estates can't be changed after creation.
- **Validation happens in the handler**, with readable messages such as `height must be between 1 and 30`. `api.yml` documents the same limits for clients, and the handler tests check each one. The request body must be exactly one JSON object: an empty body, `10.5`, `"10"` or extra data after the JSON all give 400.
- **The repository returns domain errors** (`ErrNotFound`, `ErrPlotOccupied`), never driver errors. One error handler turns every error into `{message}`. For 5xx responses the client gets a generic message and the real error is logged with the request ID.
- **Structure follows the starter**: `handler` → `repository`, oapi-codegen, gomock. There is no service layer, since it would only pass calls through. The algorithms live in two pure packages, `droneplan` and `treestats`.
- **Operability**: the DB ping is retried at startup with backoff, the connection pool is limited, the server has read/write/idle timeouts, there is a 16 KB body limit, a 25 s request timeout returns a JSON 503 and cancels the query, and SIGTERM shuts down gracefully.

## Testing

```bash
make test       # unit tests, prints coverage of hand-written code
make test_api   # API tests; needs `docker compose up --build` first
```

| Package | Coverage |
|---|---|
| `droneplan` | 100% |
| `treestats` | 100% |
| `handler` | 98.9% |
| `repository` (excluding the generated mock) | 100% |
| **Total** | **99.5%** |

- The coverage total excludes generated code (`generated/`, `*.mock.gen.go`). `cmd/main.go` is wiring only and has no unit tests.
- **`Distance` and `Rest` are checked against a brute-force simulator** that flies the estate plot by plot. The comparison runs on thousands of random estates, and for `Rest`, every possible `max_distance`. I confirmed these tests catch real mistakes by deliberately breaking the landing and neighbour rules.
- The repository is tested with go-sqlmock, the handlers with gomock.
- **API tests** (`tests/api_test.go`): 16 cases through the real stack, covering every status code, the spec example (54), an even median, the row turn, the bonus landing rules and a 50,000 × 50,000 estate.

## Assumptions

I asked the recruiter about the ambiguous points on 30 Sep, stating my defaults. The reply was to continue with the assignment, so these defaults apply.

| Question | Decision |
|---|---|
| The spec says 201 for creates; the starter test expects 200 | **201**; the test is updated |
| "404 Bad Request" for an invalid `max_distance` | 400 |
| Bonus parameter name (`max_distance`, `max-distance`, `distance`) | `max_distance` |
| Median of an even count | exact decimal (3.5) |
| Bonus `distance` | distance flown before the forced landing, `min(full plan, max_distance)` |
| Does the landing descent count toward `max_distance`? | no, it uses the backup battery |
| Malformed estate ID in the path | 400 (a well-formed but unknown ID is 404) |
| Unknown JSON fields | accepted and ignored |
| Spec mentions Go 1.22 | stays on 1.21, the grader's version |
| `/hello` example endpoint | removed |
| The README installs `go.uber.org/mock`; the starter used `github.com/golang/mock` | switched to `go.uber.org/mock` v0.4.0 |

## Known limitations

| Limitation | Notes |
|---|---|
| Drone-plan cost grows with the number of trees | Measured on one 50,000 × 50,000 estate: 1M trees 0.6 s / 70 MB, 5M trees 4.0 s / 367 MB. Around 35M trees it would hit the 25 s timeout and return 503. Scale path: store each tree's path index and keep the total distance up to date on insert (only the new tree's path neighbours change). That's O(log n) per insert and O(1) per drone-plan. |
| No auth, list/update/delete endpoints, or idempotency keys | Out of scope. A retried POST creates a duplicate estate. |
| HEAD on GET routes returns 405 | Echo doesn't map HEAD to GET. |
| A client-supplied `X-Request-Id` is trusted | Convenient for tracing across services; it means clients control that log field. |
| `docker-compose.yml` exposes Postgres on 5432 with dev credentials | For local use only. |
| No metrics or tracing | Logs include the request ID. Prometheus and OpenTelemetry would be next. |

## Project layout

```
api.yml             OpenAPI contract (code is generated into generated/)
database.sql        schema: estate, tree
cmd/main.go         config, DB connection with retry, middleware, graceful shutdown
handler/            HTTP handlers, validation, error handler, request timeout
repository/         SQL and error translation
droneplan/          patrol distance and forced-landing point (pure Go)
treestats/          count/min/max/median from a height histogram (pure Go)
tests/              API tests against the running stack
```

The Go module path is still the starter's (`github.com/SawitProRecruitment/UserService`). I kept it on purpose so imports match the original project.
