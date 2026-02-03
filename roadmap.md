# roadmap.md — Music Library Quality Scanner + Cleaner

## Phase 0 — Repo + foundations (1–3 days)
| Deliverable | Notes |
|---|---|
| Monorepo layout | /cmd/app, /internal, /web, /migrations, /artifacts |
| GOAT UI skeleton | templ layouts, Tailwind pipeline, Alpine wiring |
| DB + migrations | Libraries, MediaFiles, ScanRuns, Jobs, Tracks, Results |

## Phase 1 — Scanner MVP (3–7 days)
| Deliverable | Notes |
|---|---|
| Library registration UI | root paths, scan interval, read-only flag |
| Incremental scan engine | periodic crawl; detects new/changed via stat/size/mtime |
| Job queue (persistent) | job states: queued/running/success/fail/retry |
| Triage list | “New”, “Failed”, “Has issues” filters |

## Phase 2 — Probe + basic tests (1–2 weeks)
| Deliverable | Notes |
|---|---|
| ffprobe integration | store codec/sr/bit depth/duration/tags presence |
| Integrity checks | decode/probe failures, truncated streams |
| Basic charts | waveform envelope + peak markers (good for clipping proof) |
| Evidence export | per-track JSON export endpoint |

## Phase 3 — Lossy ancestry heuristics + spectrogram (2–4 weeks)
| Deliverable | Notes |
|---|---|
| Spectrogram artifact generation | store downsampled matrix, render client-side |
| Lossy suspicion model v1 | band-limit + rolloff + “holes” heuristics, confidence scoring |
| Per-issue “Show evidence” UI | links issue → chart + explanation |
| Album-level consistency view | spot outliers across an album |

## Phase 4 — Metadata editor + audit (2–4 weeks)
| Deliverable | Notes |
|---|---|
| Full metadata audit | missing/inconsistent tags, artwork checks |
| Safe write pipeline | atomic writes, action log, dry-run diffs |
| Bulk operations | normalize album artist, fix track/disc numbering, artwork replace |
| Optional ID lookup | add MBIDs/ISRCs with manual confirmation |

## Phase 5 — Conversion queue (2–4 weeks)
| Deliverable | Notes |
|---|---|
| Conversion profiles | iPod/Red Book compatible targets, explicit settings saved |
| Queue + worker | progress UI, logs, retry handling |
| Provenance tracking | output files link back to source + profile + timestamp |
| Post-conversion re-scan | validate outputs and attach evidence again |

## Phase 6 — Hardening + deploy UX (ongoing)
| Deliverable | Notes |
|---|---|
| Passwordless SSH deploy script | rsync binary/assets, restart systemd, health check |
| Backups + retention | DB + artifacts retention policies |
| Performance | parallel workers, throttling, NAS-friendly IO patterns |
| Security | RBAC, optional OIDC, audit log export |
