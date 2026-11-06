CREATE TABLE IF NOT EXISTS scans (
    id            TEXT PRIMARY KEY,
    target_url    TEXT NOT NULL,
    openapi_url   TEXT NOT NULL DEFAULT '',
    scan_type     TEXT NOT NULL,
    status        TEXT NOT NULL,
    progress      INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at    TEXT NOT NULL,
    updated_at    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS findings (
    id         TEXT PRIMARY KEY,
    scan_id    TEXT NOT NULL,
    rule_id    TEXT NOT NULL,
    name       TEXT NOT NULL,
    severity   TEXT NOT NULL,
    confidence TEXT NOT NULL,
    url        TEXT NOT NULL,
    evidence   TEXT NOT NULL,
    cwe        INTEGER NOT NULL DEFAULT 0,
    source     TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_findings_scan_id ON findings(scan_id);
