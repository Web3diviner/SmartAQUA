# AquaDoc Web — Development Console

Temporary frontend for developing, testing, and validating AquaDoc before it is
merged into the Smart Aqua Flutter application
(`15_AQUADOC_FRONTEND.md`).

This is a development and validation tool. It is **not** the farmer-facing
product and must not be deployed publicly.

## Scope

Implements the **First Frontend Milestone** (`15_AQUADOC_FRONTEND.md` section 22):

- **Chat** — ask a question, see a grounded structured answer
- **Farm Context Simulator** — supply simulated pond state
- **Sources** — citations with title, page, and evidence level
- **Retrieval Inspector** — what RAG actually retrieved, and how it ranked

Success criterion, from the spec:

> A developer can ask a question, optionally supply simulated farm context,
> receive a grounded AquaDoc answer, see supporting sources, see missing data,
> and inspect what RAG retrieved.

Knowledge administration, disease testing, and the evaluation screen come after
this is reliable.

## Setup

```bash
cd aquadoc-web
npm install
cp .env.example .env
npm run dev          # http://localhost:5173
```

Set `VITE_AQUADOC_DEV_TOKEN` to the same value as `AQUADOC_DEV_TOKEN` on the
service, and start AquaDoc on `:8001` first.

Expected local architecture:

```text
PostgreSQL + pgvector -> AquaDoc FastAPI :8001 -> React/Vite :5173
```

## Commands

| Command | Purpose |
|---|---|
| `npm run dev` | Development server |
| `npm run build` | Typecheck + production build |
| `npm run typecheck` | Types only |
| `npm test` | Unit tests (24) |

## Security boundaries

`15_AQUADOC_FRONTEND.md` sections 6 and 10. Browser environment variables are
public to the browser, so:

- **Never** put `LLM_API_KEY`, `EMBEDDING_API_KEY`, `DATABASE_PASSWORD`,
  `MQTT_ADMIN_PASSWORD`, or `AQUADOC_INTERNAL_SERVICE_SECRET` in a `VITE_*`
  variable. Provider keys stay on the service.
- `VITE_AQUADOC_DEV_TOKEN` is a local development credential only. AquaDoc
  refuses to start in production with a dev token configured.
- Model output is **never** rendered with `dangerouslySetInnerHTML`.
  `src/utils/markdown.tsx` parses text into React elements, so a `<script>` in
  an answer renders as literal characters. There is no HTML string to sanitize.
- Every response is validated with Zod before rendering. A payload that fails
  validation is discarded and surfaced as an error, never displayed.

## Two rules worth knowing before editing

**Empty means unknown.** Every numeric input in the simulator is a text field.
`Number('')` is `0`, so a `type="number"` input would silently turn "not
measured" into "measured as zero" — the exact failure
`04_AQUADOC_RAG_LLM.md` section 9 prohibits. Blank stays `null` all the way to
the wire, and `MeasurementValue` renders `null` as *Not available*, never `0`.
A real `0` reading still renders as `0`, because 0.0 mg/L dissolved oxygen is a
pond-killing event, not a missing value.

**Failures are named.** `15_AQUADOC_FRONTEND.md` section 15: "Do not collapse
all failures into `Something went wrong`." `AquaDocApiError.kind` distinguishes
network, retrieval, LLM, refusal, validation, timeout, and auth failures, and
each renders with its own guidance — so a retrieval outage is never mistaken
for a model outage.

## Contract stability

`src/schemas/aquadoc.ts` mirrors `aquadoc/app/schemas/chat.py`. Keeping them in
step is what makes the Flutter migration a client swap rather than a rebuild
(`15_AQUADOC_FRONTEND.md` section 19). Reuse from here: API contracts, request
and response schemas, the source model, the confidence and risk model, and the
farm-context model. Replace: the React UI, development auth, direct AquaDoc
calls, and debug-only controls.

## Long-term

This console can become the Smart Aqua Admin Console
(`15_AQUADOC_FRONTEND.md` section 21) — RAG document approval, knowledge
management, AI audit logs, prompt evaluation — while farmers use Flutter.
