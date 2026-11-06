# Documentation Standards

## Rule of Thumb

> Specification says *what the product does*.
> These docs say *how we build it*.
> If a fact is already in code, link to it. Don't copy.

---

## Where Knowledge Lives

| Kind of Knowledge | Location | Format |
| --- | --- | --- |
| Product features, field rules, business calculations | `specification.md` | Markdown, Indonesian + English |
| API contract (endpoints, request/response) | `api/openapi.yaml` | OpenAPI 3.1 |
| Architecture, layers, request flow | `architecture.md` | Markdown + ASCII diagrams |
| Coding standards, naming, patterns | `standards/*.md` | Markdown |
| Error handling rules | `error_handling.md` | Markdown |
| Development setup, Make targets | `development.md` | Markdown |
| Git branch/commit/PR conventions | `git_workflow.md` | Markdown |
| Code-specific quirk or decision | Comment next to code | `// NOTE:` or `// Why:` |

---

## English vs Indonesian

Every doc file has an Indonesian twin with same headings, tables, and order:

| English | Indonesian |
| --- | --- |
| `README.md` | `README.id.md` |
| `architecture.md` | `architecture.id.md` |
| `development.md` | `development.id.md` |
| `error_handling.md` | `error_handling.id.md` |
| `git_workflow.md` | `git_workflow.id.md` |

Rules:
- Translation, not summary.
- Same structure (headings, tables, lists).
- Code snippets stay in English (Go, SQL, JSON keys).
- Prose in natural Indonesian.

---

## Documentation Rules

### 1. One Home Per Fact

| Fact Type | Single Source |
| --- | --- |
| Product rule | `specification.md` |
| API endpoint behavior | `api/openapi.yaml` |
| Architecture decision | `docs/architecture.md` |
| Naming convention | `docs/standards/api_naming.md` |
| Code quirk | `// NOTE:` comment next to code |

**Never duplicate.** Link instead.

### 2. Link, Don't Copy

```markdown
# Good: link
See [api_naming.md](api_naming.md) for endpoint naming rules.

# Bad: copy the table
| Rule | Example |
| --- | --- |
| ... | ... |   <!-- duplicated here -->
```

### 3. Same Commit as Code

When you change behavior:
- [ ] Update `api/openapi.yaml` (if endpoints changed)
- [ ] Update relevant `docs/` file (if architecture/rules changed)
- [ ] Update tests
- [ ] All in **one commit** or **one PR**.

Example:
```bash
# Commit includes code + docs + spec
git add -A
git commit -m "feat: add load test error breakdown

- internal/dtos/loadtest_dto.go: add ErrorBreakdown field
- internal/stores/loadtest_store.go: persist error counts
- internal/services/loadtest_service.go: map breakdown to DTO
- api/openapi.yaml: document ErrorBreakdown in LoadTestResponse
- docs/architecture.md: update LoadTest flow diagram"
```

### 4. Every New Doc Linked

New doc must be linked from:
- `README.md` (start here table), OR
- `standards/standard.md` (if it's a standard)

Unlinked doc = orphan = never found.

### 5. Code Comments Point to Docs

```go
// See docs/standards/api_naming.md for endpoint naming rules.
func (h *ScanHandler) Create(w http.ResponseWriter, r *http.Request) { ... }
```

---

## Markdown Style

### Headings

```markdown
# Title (H1, once per file)
## Section (H2)
### Subsection (H3)
```

### Tables

```markdown
| Column A | Column B |
| --- | --- |
| value    | value    |
```

### Code Blocks

```go
// Go
func main() { ... }
```

```json
{ "key": "value" }
```

```sql
CREATE TABLE users (...);
```

### Links

```markdown
Relative: [architecture.md](architecture.md)
Anchor: [## Authentication](#authentication)
External: [Go Docs](https://go.dev)
```

---

## File Naming

| Doc Type | Naming | Example |
| --- | --- | --- |
| Topic guide | `{topic}.md` | `architecture.md`, `development.md` |
| Standard | `standards/{topic}.md` | `standards/api_naming.md` |
| Spec | `specification.md` | Product rules |
| Index | `README.md` | Start here |

---

## Template: New Standard Document

```markdown
# {Topic} Standards

## Overview
Brief description of why this standard exists.

## Rules
| Rule | Example | Notes |
| --- | --- | --- |
| ... | ... | ... |

## Patterns
Code examples showing correct usage.

## Anti-patterns
| Wrong | Why |
| --- | --- |
| ... | ... |

## See Also
- [Related doc](related.md)
```

---

## Updating Docs

| Change | Update |
| --- | --- |
| New endpoint | `api/openapi.yaml` + `docs/standards/api_naming.md` (if new pattern) |
| New layer/package | `docs/architecture.md` folder structure + import rules |
| New coding convention | `docs/standards/{topic}.md` + `docs/standards/standard.md` index |
| New config variable | `docs/development.md` config table |
| Bug fix in code | No doc change (unless rule was wrong) |

---

## Review Checklist

Before merging PR with doc changes:
- [ ] Linked from `README.md` or `standards/standard.md`
- [ ] Indonesian twin exists (if English changed)
- [ ] No copied content from other docs (link instead)
- [ ] Code examples compile / match actual code
- [ ] Tables have consistent columns
- [ ] No broken links (check relative paths)

---

## Doc File Inventory

| File | Purpose |
| --- | --- |
| `README.md` / `.id.md` | Start here, index, stack, folder overview |
| `architecture.md` / `.id.md` | Layers, request flow, config, startup |
| `development.md` / `.id.md` | Setup, Make targets, troubleshooting |
| `error_handling.md` / `.id.md` | AppError, wrapping, logging |
| `git_workflow.md` / `.id.md` | Branches, commits, PRs |
| `specification.md` | Product rules (single language) |
| `standards/standard.md` | Standards index + ground rules |
| `standards/api_naming.md` | Endpoint naming |
| `standards/method_naming.md` | Go method names |
| `standards/struct.md` | DTOs, struct tags |
| `standards/service_repo.md` | Layer responsibilities, DI |
| `standards/http_status.md` | Status code mapping |
| `standards/list_api.md` | Pagination, filtering |
| `standards/unit_test.md` | Testing strategy |
| `standards/documentation.md` | This file |