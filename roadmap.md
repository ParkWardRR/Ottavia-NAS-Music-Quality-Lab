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

## Phase 4 — Metadata editor + audit ✅ COMPLETE
- [x] Full metadata audit (missing/inconsistent tags, artwork checks)
- [x] Tag display in track detail UI
- [x] Artwork presence detection
- [x] Safe write pipeline (atomic writes, action log, dry-run diffs)
- [x] Bulk operations (normalize album artist, fix track/disc numbering, set fields)
- [x] Album Art Manager with extraction, upload, and bulk operations
- [x] AI-powered artwork suggestions for similar tracks
- [ ] Optional ID lookup (add MBIDs/ISRCs with manual confirmation)

## Phase 5 — Conversion queue ✅ MOSTLY COMPLETE
- [x] Conversion profiles (iPod/Red Book compatible targets)
- [x] Queue + worker infrastructure (shared with analysis jobs)
- [x] Dedicated conversion progress UI with logs
- [x] Separate output directory support (configurable per-library)
- [x] Source directory protection (read-only mode, never modifies originals)
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

## Phase 7 — Audio Scan-style Analysis ✅ COMPLETE
- [x] Analysis manifest infrastructure (analysis_manifest_v1.json per track)
- [x] Artifact standardization with versioned schemas
- [x] Audio Scan-style spectrum curve (raw msgpack.zst)
- [x] Loudness series graphs (momentary/short-term LUFS timeline)
- [x] Clipping/overs series graphs (per-window detection)
- [x] Dynamics segmentation graphs (DR/crest factor series)
- [x] Phase/correlation series graphs (stereo correlation timeline)
- [x] MessagePack + Zstd compression for raw data storage
- [x] SHA256 hashes in manifest for artifact integrity
- [x] Dynamic UI panels with Alpine.js
- [x] "Download raw data" links for each analysis module
- [x] Expected vs Detected quality classification
- [x] Bandwidth detection from spectrum analysis
- [x] DC offset detection and flagging
- [x] Nyquist and guide line computation from probe cache
- [x] Configurable analysis duration (default: first 60 seconds)
- [x] Job queue integration for audio scan jobs
- [x] Interactive/zoomable charts using raw data
- [ ] Spectrogram matrix storage with manifest pattern

## Phase 8 — Dynamic Evidence Graphs (no PNG) ✅ COMPLETE
- [x] Remove PNG artifact pipeline (no PNG generation or storage)
- [x] JSON series endpoints with decimation (`/api/tracks/{id}/audioscan/series`)
- [x] LTTB (Largest-Triangle-Three-Buckets) downsampling algorithm
- [x] Min/max envelope decimation for peak preservation
- [x] RenderHints from probe cache (nyquistHz, guideLinesHz, axis ranges)
- [x] Canvas-based interactive chart renderer (`ottavia-charts.js`)
- [x] Pan/zoom support via drag and mousewheel
- [x] Hover tooltip with crosshair
- [x] Dark/light mode theme support
- [x] Responsive canvas with high-DPI support
- [x] Guide lines for Nyquist and frequency markers
- [x] Module-specific chart panels on Track Detail page
- [x] Skeleton loading state while fetching series
- [x] Download JSON button for each chart
- [x] Reset zoom button
- [ ] Spectrogram heatmap from raw matrix (future enhancement)

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

### Output Directory Separation
- Libraries support separate output paths for conversion work
- Source directories are never modified (read-only mode)
- All conversion outputs go to user-configurable output directory
- Clean separation: `/mnt/music/Complete` (source, read-only) → `/mnt/music/Output` (conversions)

### Album Art Manager
- Dedicated album art management page
- Detect albums missing artwork at a glance
- Extract embedded artwork from audio files using FFmpeg
- Upload custom artwork with drag-and-drop interface
- Bulk apply artwork to multiple tracks simultaneously
- AI-powered suggestions for similar tracks (exact match, fuzzy match, artist match)
- Smart matching by album name and album artist with confidence scoring
- Preview changes before applying
- Real-time progress indicators for bulk operations
- Integration with existing artifact storage system

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

### Conversion Progress UI
- Full conversions page with job queue display
- Real-time progress bars for running jobs
- Status badges (queued, running, success, failed)
- Conversion profiles sidebar showing available targets
- Statistics panel with job counts by status
- "How to Convert" instructional section
- Error message display for failed jobs

### Server Deployment
- Deployed on AlmaLinux with nginx reverse proxy
- Systemd service for automatic startup
- Successfully scanning large music libraries (1500+ tracks)
- 4 worker threads for parallel analysis

### Dynamic Evidence Graphs System
- Fully dynamic, interactive browser-rendered charts (no PNG artifacts)
- JSON series API with efficient decimation (LTTB + min/max envelope)
- Canvas-based chart renderer with pan/zoom/tooltip support
- RenderHints computed from track's probe cache (never hard-coded)
- Dark/light mode support with responsive high-DPI rendering

### Audio Scan-style Analysis System
- Comprehensive audio analysis inspired by Audirvāna's Audio Scan feature
- Per-track analysis manifest (analysis_manifest_v1.json) as source of truth
- Raw data storage in MessagePack format with Zstd compression
- Dynamic charts replace static PNG images

**Spectrum Analysis Module:**
- FFT-based frequency spectrum curve generation
- Expected quality tier derived from probe cache (Hi-Res/Studio/CD Quality/Lossy)
- Detected quality classification with bandwidth analysis
- DC offset detection with configurable threshold
- Nyquist guide lines computed from track's sample rate

**Loudness Analysis Module:**
- Momentary and short-term LUFS timeline
- Integrated loudness (scalar)
- Loudness Range (LRA)
- True peak and sample peak tracking

**Clipping Detection Module:**
- Per-window clipped sample counts
- True-peak overs detection (samples exceeding 0 dBTP)
- Visual timeline of worst clipping sections

**Phase Correlation Module:**
- Stereo correlation timeline (-1 to +1)
- L/R balance tracking
- Phase issue detection (persistent negative correlation)
- Automatic skip for mono tracks

**Dynamics Segmentation Module:**
- Per-segment DR scoring
- Crest factor analysis
- Visual identification of "crushed" sections

**UI Integration:**
- Dynamic Alpine.js panels that load manifest on page view
- Each panel shows PNG chart + key metrics
- "Download raw data" links for advanced analysis
- "Run Audio Scan" button when analysis not yet performed
