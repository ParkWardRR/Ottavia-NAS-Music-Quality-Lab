# Ottavia — Music Library Quality Scanner + Cleaner

## Phase 0 — Repo + foundations (1–3 days)
- [x] Monorepo layout (`/cmd/app`, `/internal`, `/web`, `/migrations`, `/artifacts`)
- [x] GOAT UI skeleton (templ layouts, Tailwind pipeline, Alpine wiring)
- [x] DB + migrations (Libraries, MediaFiles, ScanRuns, Jobs, Tracks, Results)
- [x] Configuration system with YAML support
- [x] Logging with zerolog
- [x] Branding (Ottavia name, favicon, logo)

## Phase 1 — Scanner MVP (3–7 days)
- [x] Library registration UI (root paths, scan interval, read-only flag)
- [x] Incremental scan engine (periodic crawl; detects new/changed via stat/size/mtime)
- [x] Job queue (persistent) with states: queued/running/success/fail/retry
- [x] Triage list with "New", "Failed", "Has issues" filters
- [x] Scheduler for automatic periodic scans

## Phase 2 — Probe + basic tests (1–2 weeks)
- [x] ffprobe integration (codec/sr/bit depth/duration/tags presence)
- [x] Integrity checks (decode/probe failures, truncated streams)
- [x] Basic charts (waveform envelope + peak markers)
- [x] Evidence export (per-track JSON export endpoint)
- [x] Volume and loudness analysis

## Phase 3 — Lossy ancestry heuristics + spectrogram (2–4 weeks)
- [x] Spectrogram artifact generation (store downsampled matrix)
- [x] Lossy suspicion model v1 (band-limit + rolloff heuristics, confidence scoring)
- [x] Per-issue "Show evidence" UI (links issue → chart + explanation)
- [x] Lossy-to-lossless detection with user-friendly explanations
- [x] Dynamic Range (DR) scoring for loudness war analysis
- [x] Album page overhaul with album art, quality badges, and DR scores
- [ ] Album-level consistency view (spot outliers across an album)

## Phase 4 — Metadata editor + audit (2–4 weeks)
- [x] Full metadata audit (missing/inconsistent tags, artwork checks)
- [x] Tag display in UI
- [ ] Safe write pipeline (atomic writes, action log, dry-run diffs)
- [ ] Bulk operations (normalize album artist, fix track/disc numbering, artwork replace)
- [ ] Optional ID lookup (add MBIDs/ISRCs with manual confirmation)

## Phase 5 — Conversion queue (2–4 weeks)
- [x] Conversion profiles (iPod/Red Book compatible targets, explicit settings saved)
- [x] Queue + worker infrastructure
- [ ] Progress UI, logs, retry handling
- [ ] Provenance tracking (output files link back to source + profile + timestamp)
- [ ] Post-conversion re-scan (validate outputs and attach evidence again)

## Phase 6 — Hardening + deploy UX (ongoing)
- [x] Docker support
- [x] Makefile for build automation
- [x] Hot reload development with Air
- [x] Nginx reverse proxy configuration (port 80)
- [ ] Passwordless SSH deploy script (rsync binary/assets, restart systemd, health check)
- [ ] Backups + retention (DB + artifacts retention policies)
- [ ] Performance (parallel workers, throttling, NAS-friendly IO patterns)
- [ ] Security (RBAC, optional OIDC, audit log export)

## Testing & Documentation
- [x] Playwright-go E2E test setup
- [x] Screenshot generation for documentation
- [x] Professional README with badges
- [x] API documentation
- [ ] Unit test coverage
- [ ] Contributing guide
