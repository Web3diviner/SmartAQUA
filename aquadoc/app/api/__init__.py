"""HTTP routes.

Two surfaces, deliberately separated:

- `internal` — the stable contract the Go backend calls. Survives the Flutter
  migration unchanged.
- `dev` — temporary tooling for the React client. Mounted only outside
  production, and deleted once Flutter integration lands
  (15_AQUADOC_FRONTEND.md section 19).
"""

from app.api import dev, health, internal

__all__ = ["dev", "health", "internal"]
