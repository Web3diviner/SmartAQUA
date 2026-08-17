# Implementation Status

Tracks what has been built against `11_IMPLEMENTATION_ROADMAP.md`.

## Roadmap position

| Stage | Status | Notes |
|---|---|---|
| 1. Stabilize current product | **N/A here** | Concerns the existing Go backend, firmware, and Flutter app. Not present in this repository. |
| 2. Farm/Pond domain | **N/A here** | Go backend + Flutter work. |
| 3. Sensor normalization | **N/A here** | Go backend + firmware work. |
| 4. **AquaDoc standalone MVP** | **Built** | `aquadoc/` — see below. |
| 4b. First frontend milestone | **Built** | `aquadoc-web/` — `15_AQUADOC_FRONTEND.md` section 22. |
| 5. Farm-aware AquaDoc | **Partly built** | Farm context is accepted, reasoned over, and simulated. Go read-only integration is pending, since the backend is not in this repository. |
| 6. Disease decision support | Not started | Stage 6. |
| 7. Flutter AquaDoc UI | Not started | Stage 7. |
| 8–12 | Not started | New sensors, recommendations, commands, experts, production intelligence. |

Stages 1–3 target components that do not exist in this repository, which is
documentation-only apart from what is described here. Stage 4 is the first
buildable stage, and `00_README.md` names it the recommended first milestone.

## Stage 4 exit criteria

> Approved-document Q&A works with good grounding.

| Requirement (`04_AQUADOC_RAG_LLM.md` section 16) | Status |
|---|---|
| 1. Approved document upload | `POST /dev/v1/knowledge/documents` |
| 2. Parsing and chunking | `ingestion/parser.py`, `ingestion/chunker.py` |
| 3. Embeddings | `app/embeddings/` behind `EmbeddingProvider` |
| 4. pgvector storage | `migrations/0001_initial.sql`, HNSW cosine index |
| 5. Semantic retrieval | `app/rag/retrieval.py`, hybrid vector + lexical |
| 6. Source references | `app/rag/citations.py` |
| 7. LLM answer generation | `app/llm/` behind `LLMProvider` |
| 8. Structured output | `app/llm/schemas.py`, `ChatResponse` |
| 9. Basic farm context | `app/schemas/farm_context.py` |
| 10. Explicit missing-data notices | `missing_data` / `missing_data_labels` |

Autonomous device control is deliberately absent, per the same section.

## Non-negotiable rules, and where they are enforced

From `00_README.md`. Each has tests.

| Rule | Enforced in |
|---|---|
| LLM cannot control the feeder | `app/rules/safety.py` — actuation phrasing forced to Tier 3 |
| AquaDoc recommends; platform commands | `RecommendedAction` carries a tier and approval flag; no command type exists |
| Missing sensor data stays `unknown` | `app/schemas/farm_context.py`, `app/rules/water_quality.py` |
| AI never overwrites deterministic safety rules | `app/rules/safety.py` — guardrails only tighten |
| Health output is decision support, not diagnosis | Overclaim detection + escalation |
| Expert escalation for high-risk cases | `app/rules/safety.py` escalation triggers |
| RAG sources curated and traceable | `review_status` gate + `Provenance` |
| No secrets in source | `.env.example` only; production config validator |
| Existing feeder functionality untouched | Nothing here writes to MQTT or the device domain |

## Verification

| Check | Result |
|---|---|
| Backend tests | 132 passing |
| Frontend tests | 24 passing |
| Frontend typecheck + build | Clean |
| Cross-module import check | Clean (`aquadoc/tools/check_imports.py`) |

**Not yet verified:** retrieval SQL against a live PostgreSQL with pgvector.
No Docker or Postgres was available in the environment where this was built,
so `app/rag/retrieval.py`, the migration, and the ingestion persistence path
have not been executed against a real database. Everything the orchestrator
composes around them is covered with retrieval stubbed
(`tests/test_orchestration_end_to_end.py`).

**First thing to do on a machine with Docker:**

```bash
docker compose -f docker-compose.dev.yml up
```

then ingest and approve the sample document and ask a question — that exercises
the untested path end to end.

## Known gaps

- **Retrieval SQL is unexecuted** (above). The most likely defects are in the
  pgvector cast syntax and the array-overlap filters in `_filter_sql`.
- **Evaluation harness** — `04_AQUADOC_RAG_LLM.md` section 15 calls for a
  scored dataset measuring retrieval recall, citation correctness, and
  hallucination rate. The retrieval trace records the raw material; the dataset
  and scoring do not exist.
- **Rule thresholds are generic.** Water-quality bands and the Q10 coefficient
  in `app/rules/` are reasonable warm-water freshwater starting points, not
  calibrated values. They need review against real data before production.
- **Offline providers are stubs.** `echo` does not reason and `hashing` has no
  semantics. They exist so the pipeline runs without credentials; the config
  validator blocks both in production.
- **Chunking is untuned.** `04_AQUADOC_RAG_LLM.md` section 6 says to tune with
  retrieval evaluation rather than guessing permanently. Current values are the
  documented starting points.

## Layout

```text
aquadoc/          FastAPI service — app/ (api, llm, embeddings, rag, rules,
                  orchestration, prompts, models, schemas), ingestion/,
                  migrations/, tests/
aquadoc-web/      React + Vite development console
docker-compose.dev.yml
```
