FROM python:3.12-slim AS base

ENV PYTHONUNBUFFERED=1 \
    PYTHONDONTWRITEBYTECODE=1 \
    PIP_NO_CACHE_DIR=1

WORKDIR /srv/aquadoc

COPY pyproject.toml ./
COPY app ./app
COPY ingestion ./ingestion
COPY sample-knowledge ./sample-knowledge

RUN pip install --no-cache-dir .

# Run as an unprivileged user; the service never needs to write to its own tree.
RUN useradd --create-home --uid 10001 aquadoc
USER aquadoc

EXPOSE 8001

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
    CMD python -c "import urllib.request,sys; sys.exit(0 if urllib.request.urlopen('http://127.0.0.1:8001/internal/v1/health/live', timeout=3).status == 200 else 1)"

CMD ["sh", "-c", "uvicorn app.main:get_app --factory --host 0.0.0.0 --port ${PORT:-8001}"]
