# Ottavia — Music Library Quality Scanner + Cleaner

> Self-hosted music quality lab for audiophiles. Verify lossless authenticity, analyze dynamic range, audit metadata, and manage conversions.

## Phase 0 — Repo + foundations ✅ COMPLETE
- [x] Monorepo layout (`/cmd/app`, `/internal`, `/web`, `/migrations`, `/artifacts`)
- [x] GOAT UI skeleton (templ layouts, Tailwind pipeline, Alpine wiring)
- [x] DB + migrations (Libraries, MediaFiles, ScanRuns, Jobs, Tracks, Results)
- [x] Configuration system with YAML support
- [x] Logging with zerolog
- [x] Branding (Ottavia name, favicon, logo)
- [x] Apple-inspired glassmorphism UI design

## Phase 1 — Scanner MVP ✅ COMPLETE
- [x] Library registration UI (root paths, scan interval, read-only flag)
- [x] Incremental scan engine (periodic crawl; detects new/changed via stat/size/mtime)
- [x] Job queue (persistent) with states: queued/running/success/fail/retry
- [x] Triage list with "New", "Failed", "Has issues" filters
- [x] Scheduler for automatic periodic scans
- [x] Worker pool with configurable parallelism

## Phase 2 — Probe + basic tests ✅ COMPLETE
- [x] ffprobe integration (codec/sr/bit depth/duration/tags presence)
- [x] Integrity checks (decode/probe failures, truncated streams)
- [x] Waveform visualization with peak markers
- [x] Evidence export (per-track JSON export endpoint)
- [x] Volume and loudness analysis (LUFS, LRA, true peak)
- [x] DC offset and phase correlation analysis

## Phase 3 — Lossy ancestry heuristics + spectrogram ✅ COMPLETE
- [x] Spectrogram artifact generation (store downsampled matrix)
- [x] Lossy suspicion model v1 (band-limit + rolloff heuristics, confidence scoring)
- [x] Per-issue "Show evidence" UI (links issue → chart + explanation)
- [x] Lossy-to-lossless detection with user-friendly explanations
- [x] Dynamic Range (DR) scoring for loudness war analysis
- [x] Pro-level track analysis page with quality summary badges
- [x] Quick Assessment sidebar with Pass/Fail indicators
- [x] Visual DR scale (Crushed → Limited → Moderate → Good → Excellent)
- [x] Album page overhaul with album art, quality badges, and DR scores
- [x] Album-level consistency view (spot outliers across an album)

## Phase 4 — Metadata editor + audit ✅ MOSTLY COMPLETE
- [x] Full metadata audit (missing/inconsistent tags, artwork checks)
- [x] Tag display in track detail UI
- [x] Artwork presence detection
- [x] Safe write pipeline (atomic writes, action log, dry-run diffs)
- [x] Bulk operations (normalize album artist, fix track/disc numbering, set fields)
- [ ] Optional ID lookup (add MBIDs/ISRCs with manual confirmation)

## Phase 5 — Conversion queue 🔄 IN PROGRESS
- [x] Conversion profiles (iPod/Red Book compatible targets)
- [x] Queue + worker infrastructure (shared with analysis jobs)
- [ ] Dedicated conversion progress UI with logs
- [ ] Retry handling with exponential backoff
- [ ] Provenance tracking (output files link back to source + profile + timestamp)
- [ ] Post-conversion re-scan (validate outputs and attach evidence)

## Phase 6 — Hardening + deploy UX
- [x] Docker support (Dockerfile included)
- [x] Makefile for build automation
- [x] Hot reload development with Air
- [x] Nginx reverse proxy configuration (port 80)
- [x] Systemd service configuration
- [x] AlmaLinux production deployment tested
- [ ] Passwordless SSH deploy script (rsync binary/assets, restart systemd, health check)
- [ ] Backups + retention (DB + artifacts retention policies)
- [ ] Performance tuning (NAS-friendly IO patterns, memory optimization)
- [ ] Security hardening (RBAC, optional OIDC, audit log export)

## Testing & Documentation ✅ MOSTLY COMPLETE
- [x] Playwright-go E2E test setup
- [x] Screenshot generation with real music data
- [x] Professional README with badges and screenshots
- [x] API documentation with examples
- [x] Track detail screenshots showing pro-level analysis
- [ ] Unit test coverage (>80% target)
- [ ] Contributing guide

---

## Recent Accomplishments (Feb 2026)

### Album-Level Consistency View
- Full album detail page with artwork and metadata
- Consistency analysis showing dominant format across tracks
- Outlier detection for codec, sample rate, and bit depth
- DR score outlier detection (tracks differing >4 from album average)
- Loudness outlier detection (tracks differing >3 LUFS from average)
- Suspect track highlighting (possible transcodes)
- Quality summary sidebar with pass/check indicators
- Technical info showing dominant format
- Legend explaining outlier indicators

### Pro-Level Track Analysis Page
- Quality Summary with badges (Authenticity, Integrity, Dynamics, Clipping)
- Quick Assessment sidebar with Pass/Fail indicators
- Lossless Authenticity section with confidence scores and explanations
- Dynamic Range section with visual scale and DR scoring
- Audio Integrity & Levels with peak detection
- Technical Details section
- Visual Evidence with waveform display

### Screenshots with Real Data
- Dashboard showing 109+ tracks across multiple libraries
- Tracks page with populated music collection
- Track detail page showcasing all analysis features
- Album detail page with consistency analysis
- Dark and light mode variants
- Mobile responsive view

### Server Deployment
- Deployed on AlmaLinux with nginx reverse proxy
- Systemd service for automatic startup
- Successfully scanning large music libraries (1500+ tracks)
- 4 worker threads for parallel analysis
