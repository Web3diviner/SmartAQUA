# AquaDoc RAG and LLM Implementation Blueprint

## 1. Purpose

AquaDoc is the knowledge and conversational intelligence layer of Smart Aqua.

It should answer:

- general aquaculture questions
- farm-specific questions
- feeding questions
- water-quality questions
- disease-related questions
- historical trend questions
- "why" questions about Smart Aqua behavior

AquaDoc must remain grounded in approved sources and real farm data.

## 2. Recommended Stack

- Python 3.12+
- FastAPI
- SQLAlchemy or SQLModel
- PostgreSQL
- pgvector
- Pydantic
- external LLM API initially
- external embeddings API initially
- Redis optional
- Celery/RQ/Arq optional for ingestion workers
- object storage for source PDFs

## 3. Proposed Folder Structure

```text
aquadoc/
├── app/
│   ├── main.py
│   ├── config.py
│   ├── api/
│   │   ├── chat.py
│   │   ├── disease.py
│   │   ├── knowledge.py
│   │   └── health.py
│   ├── llm/
│   │   ├── base.py
│   │   ├── provider.py
│   │   └── schemas.py
│   ├── rag/
│   │   ├── retrieval.py
│   │   ├── filters.py
│   │   ├── reranking.py
│   │   └── citations.py
│   ├── orchestration/
│   │   ├── intent.py
│   │   ├── context_builder.py
│   │   └── orchestrator.py
│   ├── disease/
│   │   ├── assessment.py
│   │   ├── triage.py
│   │   └── escalation.py
│   ├── rules/
│   │   ├── water_quality.py
│   │   ├── feeding.py
│   │   └── safety.py
│   └── models/
├── ingestion/
│   ├── loader.py
│   ├── parser.py
│   ├── cleaner.py
│   ├── chunker.py
│   ├── metadata.py
│   ├── embedder.py
│   └── ingest.py
├── tests/
├── migrations/
├── Dockerfile
└── pyproject.toml
```

## 4. RAG Ingestion Pipeline

```text
Approved document
 -> checksum
 -> parse
 -> clean
 -> preserve page/section
 -> chunk
 -> metadata tagging
 -> embedding
 -> pgvector
 -> review status
```

### Required metadata

- source
- title
- author
- year
- page
- document type
- species
- life stage
- topic
- disease
- region
- evidence level
- review status

## 5. Evidence Levels

Recommended:

- A: official / expert-reviewed guideline
- B: peer-reviewed research
- C: established textbook/manual
- D: verified Smart Aqua expert case
- E: farmer/user report

The retriever should prefer higher evidence levels for high-risk questions.

## 6. Chunking Strategy

Initial configuration:

- target chunk size: 600-900 tokens
- overlap: 100-200 tokens
- preserve headings
- do not split tables blindly
- keep source page
- avoid chunks containing multiple unrelated topics

Tune using retrieval evaluation rather than guessing permanently.

## 7. Retrieval Pipeline

```text
question
 -> intent classification
 -> metadata filter
 -> query embedding
 -> vector similarity search
 -> lexical/hybrid search optional
 -> reranking
 -> source-quality weighting
 -> top context chunks
 -> LLM
```

## 8. Query Types

### General Aquaculture

Example:
"What is FCR?"

Context:
RAG only.

### Farm-Specific

Example:
"Why are my fish eating less today?"

Context:

- pond state
- stock/biomass
- temperature
- feeding history
- mortality
- Q10 result
- RAG

### Disease

Example:
"My fish have white lesions around their mouth."

Context:

- symptoms
- mortality
- available water data
- stock
- disease history
- images later
- disease RAG
- triage rules

## 9. Missing Data Policy

Missing values must remain explicit.

Example:

```json
{
  "temperature_c": 29.4,
  "ph": null,
  "dissolved_oxygen_mg_l": null,
  "turbidity_ntu": null
}
```

AquaDoc may say:

> pH and dissolved oxygen are not currently available, so those contributors cannot be evaluated.

It must never infer normal values.

## 10. LLM Provider Interface

Create a provider abstraction.

```python
from abc import ABC, abstractmethod

class LLMProvider(ABC):
    @abstractmethod
    async def generate(self, request):
        raise NotImplementedError
```

Implement:

```text
OpenAIProvider
AlternativeHostedProvider
LocalProvider later
```

The application should depend on `LLMProvider`, not vendor-specific APIs.

## 11. Embedding Provider Interface

Do the same for embeddings.

```python
class EmbeddingProvider:
    async def embed_documents(self, texts: list[str]) -> list[list[float]]:
        ...
    async def embed_query(self, text: str) -> list[float]:
        ...
```

## 12. Structured Responses

Do not rely on free-form text alone.

Internal response:

```json
{
  "answer": "Reduced appetite may be associated with...",
  "intent": "farm_assessment",
  "risk_level": "watch",
  "confidence": 0.78,
  "possible_causes": [
    {
      "name": "thermal stress",
      "confidence": 0.66
    }
  ],
  "recommended_actions": [],
  "missing_data": ["ph", "dissolved_oxygen"],
  "expert_escalation": false,
  "sources": []
}
```

## 13. Prompt Architecture

Keep prompts versioned.

```text
prompts/
├── general_v1.md
├── farm_assessment_v1.md
├── disease_triage_v1.md
├── feeding_explanation_v1.md
└── expert_summary_v1.md
```

Every production response should record:

- prompt version
- LLM model
- embedding model
- retrieved sources
- decision-rule version

## 14. Recommended System Rules

AquaDoc must:

- state uncertainty
- never fabricate missing measurements
- never claim laboratory confirmation
- distinguish education from case assessment
- prefer approved evidence
- avoid direct hardware commands
- escalate high-risk disease cases
- explain why a recommendation exists
- expose source references

## 15. RAG Evaluation

Build an evaluation dataset.

Example fields:

```text
question
expected_topic
expected_sources
required_facts
forbidden_facts
risk_class
```

Evaluate:

- retrieval recall
- source correctness
- factual grounding
- citation correctness
- hallucination rate
- safety/escalation correctness
- answer usefulness

## 16. First AquaDoc MVP

Must support:

1. approved document upload
2. parsing and chunking
3. embeddings
4. pgvector storage
5. semantic retrieval
6. source references
7. LLM answer generation
8. structured output
9. basic farm context
10. explicit missing-data notices

Do not build autonomous device control in the MVP.
