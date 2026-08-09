# UrlShorter

UrlShorter is a Go + React URL shortener.

- API: Go, Fiber, Postgres, Redis, Kafka, OpenTelemetry
- Web: Vite, React, TypeScript
- Local infrastructure: Docker Compose with Postgres, Redis, Kafka, and Jaeger

## Project Layout

```text
apps/api   Fiber API service
apps/web   Vite TypeScript web app
```

## Requirements

- Go 1.25+
- Node.js 24+
- Docker and Docker Compose

## Run Locally

1. Create local environment config:

```bash
cp .env.example .env
```

2. Start Postgres, Redis, Kafka, and Jaeger:

```bash
docker compose up -d
```

3. Start the API:

```bash
go run ./apps/api/cmd
```

The API runs at `http://localhost:8080`.

4. Start the web app:

```bash
cd apps/web
npm install
npm run dev
```

The web app runs at `http://localhost:5173`.

5. Open observability UI:

```text
http://localhost:16686
```

## Useful Commands

```bash
go test $(go list ./... | grep -v '/node_modules/')
```

```bash
cd apps/web
npm run lint
npm run build
```

## API Endpoints

- `GET /healthz` checks API health.
- `POST /api/links` creates a short link.
- `GET /api/links` lists recent links.
- `GET /api/links/{code}` returns one link.
- `DELETE /api/links/{code}` soft-deletes one link by marking it expired.
- `GET /{code}` redirects and publishes a Kafka click event.

New links expire after `DEFAULT_LINK_TTL`, which defaults to `168h` or 7 days. Expired or soft-deleted links return `404 link not found` when clicked and are hidden from the link list used by the web analytics view.

Example:

```bash
curl -X POST http://localhost:8080/api/links \
  -H "Content-Type: application/json" \
  -d '{"targetUrl":"https://github.com/agungardiyanta/UrlShorter","customCode":"repo"}'
```

## Environment Variables

See `.env.example` for all supported values.

Important values:

- `DATABASE_URL`: Postgres connection string.
- `REDIS_ADDR`: Redis host and port.
- `KAFKA_BROKERS`: comma-separated Kafka brokers.
- `KAFKA_CLICK_TOPIC`: Kafka topic for redirect/click events.
- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry collector endpoint.
- `BASE_URL`: public base URL used when returning short links.
- `FRONTEND_ORIGIN`: allowed browser origin for CORS.
- `DEFAULT_LINK_TTL`: default lifetime for new links, for example `24h`, `168h`, or `720h`.

## Docker Images

Build API:

```bash
docker build -f apps/api/Dockerfile -t urlshorter-api .
```

Build web:

```bash
docker build -f apps/web/Dockerfile -t urlshorter-web .
```

## GitHub Actions

The repository has:

- `ci.yaml`: tests the Go API and lints/builds the Vite app.
- `api-build-pipeline.yaml`: builds and pushes the API image.
- `web-build-pipeline.yaml`: builds and pushes the web image.
- `build.yaml`: reusable Docker build and deployment-manifest update workflow.

Deployment manifest updates target [UrlShorterDeployment](https://github.com/agungardiyanta/UrlShorterDeployment).
