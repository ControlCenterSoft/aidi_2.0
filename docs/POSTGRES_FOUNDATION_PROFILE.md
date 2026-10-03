# PostgreSQL 16 foundation profile

Release A uses a repository-owned PostgreSQL development/test profile at
`deploy/dev/postgres.compose.yml`.

The profile intentionally:

- pins PostgreSQL to `postgres:16.12-bookworm`;
- creates the disposable `aidi` database and `aidi` development user;
- stores data on a tmpfs so every CI start is clean;
- exposes a container health check based on `pg_isready`;
- publishes no database port to the host;
- uses a development/test-only password and is not a production deployment profile.

## GitHub CI evidence

The `postgres` job in `.github/workflows/ci.yml` validates the rendered
Compose configuration, starts the profile with Docker Compose, waits for the
container health check, and verifies the running server reports PostgreSQL
16.12.

The job always exports container state/logs on failure and tears the profile
down afterward. This profile is for GitHub-hosted development/test validation
only; it does not connect to or mutate any existing AIDI database, VM, or
local development environment.
