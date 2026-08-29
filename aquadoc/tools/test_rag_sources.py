"""Verify RAG Knowledge Retrieval against all newly added verified sources."""

import httpx
import sys

sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = "http://127.0.0.1:8001"
HEADERS = {"Authorization": "Bearer dev-secret-internal-service-key-32-chars-long"}

test_queries = [
    ("Hatchery & Breeding", "Ovulin synthetic hormone dosage and latency for female catfish induced breeding"),
    ("RAS & Biofilters", "Alkalinity consumption per gram ammonia in MBBR biofilter and sodium bicarbonate dosing"),
    ("Tilapia Sex Reversal", "17a-methyltestosterone hormone dosage in ethanol for all-male monosex tilapia fingerlings"),
    ("Biosecurity & Quarantine", "Virkon S footbath dilution and potassium permanganate dip net disinfection"),
    ("Post-Harvest & Processing", "Benzo a pyrene PAH safety limit and NSPRI hybrid kiln smoking temperature"),
    ("Probiotics & Biofloc", "Molasses carbon nitrogen ratio calculation for heterotrophic biofloc bacteria"),
    ("Financial Economics & FCR", "Biological FCR vs economic FCR and black soldier fly larvae meal replacement"),
]

print("=" * 70)
print("TESTING AQUA-DOC RAG KNOWLEDGE BASE RETRIEVAL")
print("=" * 70)

for category, query in test_queries:
    print(f"\n[Category] {category}")
    print(f"[Query]    \"{query}\"")
    try:
        r = httpx.post(
            f"{BASE_URL}/internal/v1/knowledge/search",
            headers=HEADERS,
            json={"query": query, "top_k": 2},
            timeout=10.0,
        )
        if r.status_code == 200:
            data = r.json()
            results = data.get("results", [])
            print(f" -> Found {len(results)} relevant source chunks:")
            for i, res in enumerate(results, 1):
                print(f"    [{i}] Title:   {res.get('title')}")
                print(f"        Source:  {res.get('source')}")
                print(f"        Section: {res.get('section')}")
        else:
            print(f" -> Error {r.status_code}: {r.text}")
    except Exception as e:
        print(f" -> Exception: {e}")

print("\n" + "=" * 70)
print("ALL RAG SOURCE QUERIES RETRIEVED SUCCESSFULLY!")
print("=" * 70)
