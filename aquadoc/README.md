# AquaDoc Service

The knowledge and decision-support layer of Smart Aqua
(`01_SYSTEM_ARCHITECTURE.md` section 4).

Implements **Stage 4 — AquaDoc Standalone MVP** from
`11_IMPLEMENTATION_ROADMAP.md`.

## What this service is

AquaDoc answers aquaculture questions grounded in approved knowledge documents,
optionally in the context of a specific pond.

It is decision support. It is **not** a veterinarian, a laboratory, or a device
controller. It produces recommendations; the Go backend produces commands, and
only after a human approves them.

## Architecture

```text
   api            internal/v1 (Go backend)   dev/v1 (temporary web client)
    |
 orchestrator     intent -> rules -> retrieval -> prompt -> LLM -> safety
    |
 retrieval / rules / models
    |
 providers        LLMProvider, EmbeddingProvider (vendor code lives only here)
```

One chat turn runs in one transaction: conversation, user message, assistant
message, and retrieval trace all persist together or not at all.

## Requirements

- Python 3.12+
- PostgreSQL 16 with the `pgvector` extension

## Setup

```bash
cd aquadoc
python -m venv .venv && source .venv/bin/activate    # Windows: .venv\Scripts\activate
pip install -e ".[dev]"

cp .env.example .env         # then edit — never commit .env
```

Apply the schema:

```bash
psql "$DATABASE_URL" -f migrations/0001_initial.sql
```

Run:

```bash
uvicorn app.main:get_app --factory --reload --port 8001
```

Or bring up the whole local stack (Postgres + service + web client):

```bash
docker compose -f ../docker-compose.dev.yml up
```

## Running without any API key

The default configuration uses two offline providers so the pipeline runs end
to end with no credentials and no network:

| Setting | Development default | What it is |
|---|---|---|
| `LLM_PROVIDER` | `echo` | Extractive stub. Quotes retrieved passages verbatim; does **not** reason. |
| `EMBEDDING_PROVIDER` | `hashing` | Hashed character n-grams. Lexical overlap only; no semantics. |

Both are development scaffolding. `Settings` **refuses to start** with either of
them when `APP_ENV=production`, so they cannot ship by accident.

For real answers:

```bash
LLM_PROVIDER=claude
ANTHROPIC_API_KEY=sk-...
EMBEDDING_PROVIDER=voyage
VOYAGE_API_KEY=...
```

## Ingesting knowledge

```bash
python -m ingestion.ingest \
  --file docs/fao-aquaculture-manual.pdf \
  --title "FAO Aquaculture Manual" \
  --source "FAO" \
  --document-type guideline \
  --evidence-level A \
  --species "Clarias gariepinus" \
  --topic feeding --topic water_quality
```

Documents land in `pending`. **Ingesting is not approving.** Nothing is
retrievable until a human approves it:

```bash
curl -X POST http://localhost:8001/dev/v1/knowledge/documents/<id>/approve \
  -H "Authorization: Bearer $AQUADOC_DEV_TOKEN"
```

Evidence levels (`04_AQUADOC_RAG_LLM.md` section 5): `A` official guideline,
`B` peer-reviewed, `C` textbook, `D` verified expert case, `E` user report.
Higher levels are preferred for high-risk questions.

## API

| Surface | Prefix | Caller | Lifetime |
|---|---|---|---|
| Internal | `/internal/v1` | Go backend | Permanent — stable contract |
| Developer | `/dev/v1` | Temporary web client | Removed after Flutter integration |
| Health | `/health` | Probes | Permanent |

The developer surface is mounted **only** when `APP_ENV=development` and
`AQUADOC_DEV_TOKEN` is set, so it does not exist in production.

Interactive docs at `/docs` in non-production environments.

## Design rules enforced in code

These come from `00_README.md` and `14_AQUADOC_SAFETY_AND_GOVERNANCE.md`, and
each has tests:

| Rule | Where |
|---|---|
| Missing data stays `unknown`, never assumed normal | `schemas/farm_context.py`, `rules/water_quality.py` |
| AI output never overrides deterministic rules | `rules/safety.py` — guardrails only tighten |
| AquaDoc never issues device commands | `rules/safety.py` — actuation phrasing forced to Tier 3 |
| Only approved documents are retrieved | `rag/filters.py` + re-asserted in `rag/retrieval.py` SQL |
| Confidence is not LLM self-confidence | `orchestration/confidence.py` — model is 15% of the score |
| Citations cannot be fabricated | `rag/citations.py` — built from retrieval rows |
| Retrieved text is untrusted | `orchestration/context_builder.py` |
| Every response records provenance | `schemas/chat.py::Provenance` |

## Tests

```bash
python -m pytest              # 132 tests
python tools/check_imports.py # static cross-module import check
```

Retrieval SQL requires a live PostgreSQL with pgvector and is not covered by the
unit suite; `tests/test_orchestration_end_to_end.py` covers everything the
orchestrator composes with retrieval stubbed.

> On Python 3.14 the suite passes but prints a benign access violation from
> `anyio` during `TestClient` teardown. Use 3.12 to avoid the noise.

## Not built yet

Deliberately out of scope for Stage 4, in roadmap order:

- Disease decision support and `DiseaseCase` (Stage 6)
- Recommendation persistence and approval (Stage 9)
- Command linking (Stage 10)
- Expert consultation (Stage 11)
- Evaluation harness (`04_AQUADOC_RAG_LLM.md` section 15) — the retrieval trace
  provides the raw material, but the scored dataset does not exist yet
