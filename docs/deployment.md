# Deployment Notes

## Live URLs

- Frontend: <https://phifin.github.io/keyloop-service-scheduler/>
- Backend API: <https://keyloop-scheduler-api.onrender.com>
- Database: Supabase PostgreSQL

The live deployment is intended for assessment review and demo recording. The local setup remains the reliable source of truth for validating migrations, seed data, tests, and backend behavior.

## Frontend Deployment

The frontend is a Vite React app deployed to GitHub Pages.

Important details:

- Build output: `frontend/dist`
- Router: `HashRouter`, used so GitHub Pages refreshes do not return 404.
- GitHub Pages base path: `/keyloop-service-scheduler/`
- Production API URL: `https://keyloop-scheduler-api.onrender.com`

GitHub Actions build environment:

```sh
VITE_BASE_PATH=/keyloop-service-scheduler/
VITE_API_BASE_URL=https://keyloop-scheduler-api.onrender.com
```

The workflow is defined in `.github/workflows/deploy-frontend.yml` and deploys `frontend/dist` using GitHub Pages actions.

## Backend Deployment

The backend API is deployed on Render.

Required Render environment variables:

```sh
PORT=8080
DATABASE_URL=<supabase-postgresql-connection-string>
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://phifin.github.io
```

Render provides the actual listening port through `PORT`. The Go server reads `PORT`, `DATABASE_URL`, and `CORS_ALLOWED_ORIGINS` from the environment.

Render free-tier services may sleep after inactivity. The first request after a cold period can take 30-60 seconds while the backend wakes up.

## Database Deployment

The deployed database is Supabase PostgreSQL.

Notes:

- The schema is managed by the SQL migration files under `backend/db/migrations`.
- Demo records are managed by `backend/db/seed.sql`.
- Supabase Session Pooler may be needed when deploying from IPv4-only networks.
- Production secrets should be configured in Render environment variables, not committed to the repository.

## CORS

The backend uses an explicit comma-separated origin allowlist. It compares the request `Origin` header against the configured origins and echoes the matched origin in `Access-Control-Allow-Origin`.

The deployed frontend requires:

```sh
CORS_ALLOWED_ORIGINS=https://phifin.github.io
```

For local and deployed use together:

```sh
CORS_ALLOWED_ORIGINS=http://localhost:5173,https://phifin.github.io
```

Wildcard CORS is intentionally not used.

## Local Production Build Check

To reproduce the GitHub Pages build locally:

```sh
cd frontend
VITE_BASE_PATH=/keyloop-service-scheduler/ VITE_API_BASE_URL=https://keyloop-scheduler-api.onrender.com npm run build
```
