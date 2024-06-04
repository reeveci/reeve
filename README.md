# Reeve CI / CD

Simple extensible open source CI/CD solution written in Go.

## Common Concepts

- Plugins should implement common settings (which can be configured for all plugins using `REEVE_SHARED_` scope):
  - `ENABLED=true`
  - `TRUSTED_DOMAINS`
  - `TRUSTED_TASKS` (split by spaces -> strings.Fields)
  - `SETUP_GIT_TASK`

## Roadmap

- Metrics
  - Status Badges
  - Display queued pipelines / more details
