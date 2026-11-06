# Git Workflow & Contributing

## Branch Model

- `main` / `master`: Production release branch. Clean, stable, deployable.
- `develop`: Integration branch. All features and bugfixes branch off and merge here.
- `feat/*`: Feature branch (e.g. `feat/auth-redis`).
- `fix/*`: Bugfix branch (e.g. `fix/loadtest-duration`).
- `docs/*`: Documentation additions or updates (e.g. `docs/architecture`).

---

## Commit Message Convention

Conventional Commits format:

```
<type>(<scope>): <short summary in imperative mood>

[optional body explaining why and what changed]

[optional issue reference]
```

### Types

| Type | When to Use |
| --- | --- |
| `feat` | New user-facing or API functionality |
| `fix` | Bug fix in code or behavior |
| `docs` | Documentation updates only |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `test` | Adding or updating unit tests |
| `chore` | Dependency updates, tooling, config changes |

### Example

```
feat(auth): add redis refresh token cache and jwt middleware

- Implement RedisStore for token storage, revocation blacklist, and sessions
- Add JWT HS256 access token (15m) and refresh token (7d) helpers
- Add SQLite user_store with bcrypt password hashing
- Add auth HTTP middleware protecting scans and load-tests endpoints
- Add first-run admin account setup flow
```

---

## Pull Request Rules

1. **Keep PRs focused:** One concern per PR (e.g., Auth feature or Load test fix, not both).
2. **Build & tests pass:** `go test ./...` and `go build ./...` must succeed with zero errors.
3. **Docs included:** Any endpoint or behavior change must update documentation and OpenAPI spec in the same PR.
4. **No secrets:** Ensure no passwords, real JWT secrets, or internal URLs are committed.
