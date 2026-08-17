# AquaDoc Temporary Frontend Architecture and Implementation

## 1. Purpose

This document defines the temporary frontend used to develop, test, validate, and refine AquaDoc before AquaDoc is merged into the existing Smart Aqua Flutter application.

The temporary frontend is a development and validation tool. It is not intended to replace the existing farmer-facing mobile application.

The target production flow remains:

```text
Smart Aqua Flutter App
        |
        v
Existing Go Backend
        |
        v
AquaDoc Python Service
        |
   +----+----+
   |    |    |
  RAG  Rules LLM
```

During development, use:

```text
AquaDoc Web Frontend
        |
        v
AquaDoc FastAPI
        |
   +----+----+
   |    |    |
  RAG  Rules LLM
```

The key rule is to keep the AquaDoc API contract stable so the temporary web frontend can later be replaced by the Flutter integration without rebuilding the RAG or LLM system.

## 2. Recommended Stack

Use:

- React
- TypeScript
- Vite
- React Router
- TanStack Query or a small centralized API layer
- Zod for runtime schema validation
- Tailwind CSS or a lightweight component system
- Vitest
- React Testing Library
- Playwright later for end-to-end tests

Repository structure:

```text
Project/
├── backend/
├── firmware/
├── mobile/
├── aquadoc/
└── aquadoc-web/
```

Suggested frontend structure:

```text
aquadoc-web/
├── src/
│   ├── app/
│   │   ├── router.tsx
│   │   └── providers.tsx
│   ├── pages/
│   │   ├── ChatPage.tsx
│   │   ├── FarmSimulatorPage.tsx
│   │   ├── KnowledgeBasePage.tsx
│   │   ├── DiseaseTestPage.tsx
│   │   ├── EvaluationPage.tsx
│   │   └── SettingsPage.tsx
│   ├── components/
│   │   ├── ChatMessage.tsx
│   │   ├── SourceCard.tsx
│   │   ├── MissingDataBadge.tsx
│   │   ├── ConfidenceBadge.tsx
│   │   ├── RiskBadge.tsx
│   │   ├── RetrievalInspector.tsx
│   │   ├── FarmContextPanel.tsx
│   │   └── ErrorBoundary.tsx
│   ├── api/
│   │   ├── client.ts
│   │   ├── chat.ts
│   │   ├── knowledge.ts
│   │   ├── disease.ts
│   │   └── evaluation.ts
│   ├── schemas/
│   ├── types/
│   ├── hooks/
│   ├── utils/
│   └── main.tsx
├── public/
├── tests/
├── .env.example
├── package.json
├── tsconfig.json
└── vite.config.ts
```

## 3. Development Objectives

The temporary frontend must let the development team verify that:

1. AquaDoc receives questions.
2. AquaDoc returns structured responses.
3. RAG retrieval returns relevant passages.
4. Sources and page references are correct.
5. Missing farm data is disclosed.
6. Farm context materially changes contextual answers.
7. Disease workflows behave safely.
8. Confidence and risk indicators are visible.
9. Prompt/model versions can be inspected in developer mode.
10. Retrieval failures are distinguishable from LLM failures.
11. AquaDoc can be tested without changing the live Flutter application.

## 4. Required Screens

### Chat

Primary AquaDoc testing interface.

```text
+---------------------------------------------------+
| AquaDoc                                           |
+---------------------------------------------------+
| Context: General Aquaculture                      |
|                                                   |
| User: What is FCR?                                |
|                                                   |
| AquaDoc: Feed conversion ratio is...              |
|                                                   |
| Sources                                           |
| - Source A, page 14                               |
| - Source B, page 22                               |
|                                                   |
| Confidence: High                                  |
| Risk: Informational                               |
| Missing Data: None                                |
+---------------------------------------------------+
| Ask AquaDoc...                              Send  |
+---------------------------------------------------+
```

Required features:

- send question
- loading state
- retry
- source cards
- confidence display
- risk display
- missing-data display
- conversation reset
- developer details toggle

### Farm Context Simulator

Before the Go backend supplies real pond context, developers should be able to simulate it.

Inputs:

- farm name
- pond name
- species
- population
- average weight
- biomass
- temperature
- pH
- dissolved oxygen
- turbidity
- daily ration
- last feeding
- mortality in last 24 hours
- symptoms

Current project defaults should reflect reality:

```json
{
  "temperature_c": 29.4,
  "ph": null,
  "dissolved_oxygen_mg_l": null,
  "turbidity_ntu": null
}
```

The UI must render unavailable measurements as `Not available`, never `0`.

### Knowledge Base

Development/admin only.

Functions:

- list documents
- filter by status
- filter by species
- filter by topic
- inspect metadata
- inspect chunk count
- inspect ingestion status
- approve
- reject
- deprecate

Statuses:

```text
pending
approved
deprecated
rejected
```

Only approved knowledge should be used in production RAG.

### Disease Assessment Tester

Inputs:

- species
- growth stage
- number affected
- symptoms
- duration
- mortality
- available environmental data
- images later

Output:

- possible conditions
- confidence
- supporting evidence
- alternatives
- missing information
- next steps
- expert escalation status

Use labels such as `AquaDoc Assessment` or `Possible Conditions`, not `Confirmed Diagnosis`, unless a verified expert or laboratory result exists.

### Evaluation / Debug

Show:

- question
- intent
- metadata filters
- retrieved chunks
- similarity scores
- reranking scores
- evidence level
- source title
- page number
- selected context
- latency
- token usage
- estimated cost
- prompt version
- model version

This screen is for developers, not farmers.

## 5. Farm-Aware Chat Modes

Context selector:

```text
General Aquaculture
Simulated Pond
```

Later:

```text
General Aquaculture
Farm A / Pond 1
Farm A / Pond 2
```

General:

```text
Question
 -> RAG
 -> LLM
```

Farm-aware:

```text
Question
 + Farm Context
 + RAG
 + Rules
 -> LLM
```

## 6. API Client

Use a centralized frontend API client.

Development environment:

```bash
VITE_AQUADOC_API_URL=http://localhost:8001
VITE_APP_ENV=development
VITE_ENABLE_DEBUG_PANEL=true
```

Never expose these in frontend environment variables:

```text
LLM_API_KEY
EMBEDDING_API_KEY
DATABASE_PASSWORD
MQTT_ADMIN_PASSWORD
AQUADOC_INTERNAL_SERVICE_SECRET
```

Browser environment variables are public to the browser.

## 7. Development Endpoints

Temporary direct-to-AquaDoc development endpoints can include:

```text
POST /dev/v1/chat
POST /dev/v1/disease/assess
POST /dev/v1/farm/analyze

GET  /dev/v1/knowledge/documents
POST /dev/v1/knowledge/documents
POST /dev/v1/knowledge/documents/{id}/approve
POST /dev/v1/knowledge/documents/{id}/reject

GET  /dev/v1/debug/retrieval/{request_id}
POST /dev/v1/evaluations/run
```

These endpoints must be development/admin restricted and should not become public farmer APIs.

## 8. Final Production Boundary

Final farmer flow:

```text
Flutter
    |
    v
Go Backend
    |
    v
AquaDoc
```

The Go backend remains responsible for:

- authentication
- authorization
- farm ownership
- pond permission checks
- rate limits
- audit logging
- recommendation approval
- device command creation

AquaDoc should not become a second public authentication system.

## 9. Development Authentication

### Development Token

Suitable for earliest local testing.

Frontend sends:

```text
Authorization: Bearer <development-token>
```

Requirements:

- only enabled in development
- not committed
- impossible to use in production

### Existing Go Authentication

Use this for integration testing after the standalone AquaDoc is stable.

```text
React
 -> Go Backend
 -> AquaDoc
```

This verifies the same boundary the Flutter app will use.

## 10. Frontend Security

The temporary frontend must:

- never contain provider secrets
- sanitize rendered Markdown
- disallow unrestricted raw HTML
- validate API responses
- protect against XSS
- enforce upload-size limits
- avoid exposing stack traces
- avoid logging sensitive tokens
- use HTTPS when deployed remotely

Do not render LLM content using unrestricted `dangerouslySetInnerHTML`.

## 11. Response Validation

Use Zod or equivalent runtime validation.

Example:

```ts
const AquaDocResponse = z.object({
  answer: z.string(),
  intent: z.string(),
  confidence: z.number().min(0).max(1),
  risk_level: z.string(),
  missing_data: z.array(z.string()),
  sources: z.array(
    z.object({
      title: z.string(),
      page: z.number().nullable()
    })
  )
});
```

Invalid backend responses should fail safely.

## 12. Source Presentation

Every grounded answer should display supporting sources.

Example:

```text
Sources

FAO Aquaculture Manual
Page 48

African Catfish Production Guide
Page 22
```

Developer mode may also show the actual retrieved chunk and retrieval score.

## 13. Confidence UI

Do not imply false scientific precision to farmers.

Farmer-facing:

```text
Low
Moderate
High
```

Developer-facing numeric value may still be available.

Example initial mapping:

```text
0.00-0.49 = Low
0.50-0.74 = Moderate
0.75-1.00 = High
```

Tune later using evaluation data.

## 14. Missing Data UI

For contextual questions, clearly show what AquaDoc could not evaluate.

Example:

```text
Data not currently available

- pH
- Dissolved Oxygen
- Turbidity
```

This is important because temperature is currently the active water-quality measurement while the other sensors have not yet been installed.

## 15. Error States

Distinguish:

- frontend cannot reach AquaDoc
- RAG retrieval failed
- LLM provider failed
- response validation failed
- upload failed
- knowledge document not approved
- context incomplete

Do not collapse all failures into `Something went wrong`.

## 16. Local Setup

```bash
cd aquadoc-web
npm install
cp .env.example .env
npm run dev
```

Expected local architecture:

```text
PostgreSQL + pgvector
        |
AquaDoc FastAPI :8001
        |
React/Vite :5173
```

Later include:

```text
Go Backend :8080
```

for integration tests.

## 17. Docker Development

A future `docker-compose.dev.yml` can include:

```text
postgres
redis
aquadoc
aquadoc-worker
aquadoc-web
```

The existing Go backend can then be added for full integration.

## 18. Frontend Tests

### Unit

- response rendering
- source cards
- confidence labels
- missing-data display
- schema validation

### Integration

- chat request
- knowledge list
- document upload
- disease assessment

### End-to-End

```text
Ask question
 -> response appears
 -> sources appear
 -> missing data appears
 -> retrieval details available in developer mode
```

Use Playwright later.

## 19. Migration Into Flutter

Do not copy React components into Flutter.

Reuse:

- API contracts
- request schemas
- response schemas
- source model
- confidence/risk model
- conversation behavior
- farm-context model
- disease workflow
- evaluated UX decisions

Replace:

- React UI
- development auth
- direct AquaDoc calls
- debug-only controls

Final path:

```text
Flutter
 -> Go Backend
 -> AquaDoc
```

## 20. Recommended Flutter Migration Order

1. Add AquaDoc method to existing Flutter `ApiService`.
2. Add AquaDoc response models.
3. Add Riverpod AquaDoc provider.
4. Add AquaDoc chat screen.
5. Add source cards.
6. Add context selector.
7. Connect real Go farm/pond context.
8. Add disease workflow.
9. Add recommendations.
10. Keep the web frontend as an internal admin/debug console if useful.

## 21. Long-Term Use of the Web Frontend

The temporary frontend can evolve into:

```text
Smart Aqua Admin Console
```

Possible functions:

- RAG document approval
- knowledge management
- expert management
- AquaDoc evaluation
- AI audit logs
- support cases
- device fleet monitoring
- prompt/model evaluation

This gives the temporary frontend long-term value while farmers continue using Flutter.

## 22. First Frontend Milestone

Build only:

```text
Chat
Farm Context Simulator
Sources
Retrieval Inspector
```

Success criterion:

> A developer can ask a question, optionally supply simulated farm context, receive a grounded AquaDoc answer, see supporting sources, see missing data, and inspect what RAG retrieved.

Only after this is reliable should knowledge administration, disease testing, and broader admin functions be added.
