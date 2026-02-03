# spec.md — Music Library Quality Scanner + Cleaner (VM/NAS)

# product name:     Ottavia-NAS-Music-Quality-Lab (feel free to shorten)

## 1) Product summary
A self-hosted tool (runs in a VM) that monitors one or more NAS-mounted music folders, scans new/changed audio, verifies “true lossless” vs likely lossy-transcodes, runs a battery of audio-quality tests, audits metadata, and provides per-track interactive charts as evidence. It also provides a web UI to edit metadata and to queue album/track conversion jobs for “iPod-max / Red Book style” targets, executed later inside the app.

Primary outcomes:
- Trust: give defensible evidence for “this FLAC is (probably) real lossless” vs “likely lossy→FLAC.”
- Hygiene: catch broken/corrupt files, clipping, weird sample rates, mismatched track/album metadata, missing tags/artwork.
- Workflow: triage → fix metadata in UI → queue conversions → process later → re-scan and prove.

## 2) Target environment & constraints
| Item | Requirement |
|---|---|
| Runtime | Go backend on AlmaLinux host (VM), deployed via passwordless SSH |
| Storage | NAS mounted into VM (SMB/NFS). Tool must tolerate mounts that don’t reliably emit inotify events |
| UI | GOAT stack: Go + templ + Alpine.js + Tailwind; charts via SciChart.js (preferred) |
| UI fallback | Next.js (React + TypeScript) + ApexCharts or Recharts, backed by the same Go APIs |
| Ops | Designed for long-running background jobs with resumable scanning and idempotent processing |
| Safety | Never modifies originals unless explicitly enabled per-library; default is read-only scanning |

## 3) Core workflows (end-to-end)
### 3.1 Ingest / monitor
| Step | Behavior |
|---|---|
| Library registration | User adds one or more “Libraries” (root folders) and selects policies (scan interval, read-only, allowed formats, output folders) |
| Change detection | Default: periodic crawl + fast stat/size/mtime check, optional content hashing for suspicious cases; store per-file fingerprint |
| Job creation | New/changed files become “Analysis jobs”; failures create retryable jobs with exponential backoff |

### 3.2 Audio analysis pipeline (per file)
The pipeline produces:
- Machine-readable results (JSON), persisted to DB
- Evidence artifacts (images/arrays) for charts
- A “Confidence score” that explicitly separates proof vs heuristic suspicion

| Stage | Output | Notes |
|---|---|---|
| Container & stream probe | codec, sample rate, bit depth, channels, duration, tags presence | via `ffprobe` integration |
| Decode to PCM (analysis only) | deterministic analysis frames | done in chunks; avoid full-file RAM load |
| Integrity checks | decode errors, truncated frames, block CRC (when available), duration anomalies | file-specific strategies (FLAC vs others) |
| “Likely lossy→lossless” heuristics | band-limit detection, high-frequency rolloff patterns, spectral “holes,” suspicious encoder artifacts | outputs: {pass/warn/fail}+score+explanations |
| Clipping / headroom | peak stats, clipped-sample counts, true-peak approximation, crest factor | flag “hard clipped” and “likely limited” |
| Noise / DC offset | DC offset estimate, low-frequency rumble hints | warnings only; don’t “fix” automatically |
| Stereo sanity | phase correlation, swapped channels suspicion | warnings + chart overlays |
| Loudness stats | approximate integrated loudness + short-term distribution | for consistency checks within an album |
| Duplicate detection | audio fingerprint + duration tolerance + hash | identify same audio across different files |

### 3.3 Evidence charts (per track)
Evidence is not just a score: every warning must link to a chart view and raw numbers.

| Chart | Evidence shown | Storage |
|---|---|---|
| Spectrogram (STFT) | frequency content over time; used for band-limit/transcode suspicion | store downsampled matrix (e.g., 512–2048 bins) |
| HF energy plot | energy above thresholds (e.g., >16kHz, >19kHz) over time | store series arrays |
| Waveform + peak markers | clipped regions, peaks, quiet vs loud sections | store min/max envelope per block |
| Album consistency | loudness/peak distribution across tracks; outliers highlighted | derived from per-track stats |

UI requirements:
- “Show evidence” button for every issue
- Export evidence bundle per track: JSON + PNG (or WebP) images + a short textual explanation

### 3.4 Metadata audit + UI editor
| Category | Checks | Fix/assist |
|---|---|---|
| Required tags | track title, track #, album, album artist, artist, date/year, disc # (if multi-disc), genre (optional policy) | inline editor with validation |
| Consistency | track numbering gaps, mixed album artist spellings, inconsistent year, multiple values formatting | “Normalize” suggestions with diff preview |
| Artwork | missing cover, tiny cover, weird aspect ratio | allow upload/replace, store original + resized |
| ReplayGain / R128 | missing or inconsistent | optional “calculate and write” per-library policy |
| Music IDs | missing MBIDs/ISRCs | optional lookup flow (manual confirm before writing) |

Editing safety:
- “Dry run” mode shows tag diffs without writing
- Writes are atomic per-file (write to temp + rename where feasible)
- Every write is logged as an “Action” with before/after snapshot

### 3.5 Conversion queue (iPod / Red Book targets)
Goal: flag albums/tracks to be converted later by an internal job queue.
- Policy: select “iPod target” per playlist/album/library
- Enqueue: user chooses destination root (separate tree from originals)
- Execution: run converter pipelines as background jobs with progress + logs
- Post-step: re-scan outputs and attach provenance (“converted from X by profile Y at time T”)

Conversion profiles (initial):
- “iPod Max Compatibility”: 16-bit / 44.1kHz, plus a codec target (ALAC/AAC) depending on your preference
- “Red Book normalize” (optional): resample + dither strategy (explicitly documented), no loudness normalization unless enabled

## 4) System architecture
### 4.1 Services (single-node)
| Component | Responsibility | Tech |
|---|---|---|
| Web app | UI pages + API | Go + templ; JSON APIs |
| Scanner | periodic library crawling + job creation | Go workers |
| Analyzer | runs probe/decoding/analysis, stores results/artifacts | Go workers + external binaries |
| Converter | runs conversion jobs; stores outputs and provenance | Go workers + external pipelines |
| DB | persistence for libraries, files, results, actions | PostgreSQL preferred; SQLite acceptable for MVP |
| Artifact store | spectrogram matrices, derived images, logs | local disk path; later S3-compatible optional |

### 4.2 External tools strategy
Principle: use stable external binaries for probing/decoding/resampling unless Go-native libraries are clearly sufficient and maintained.

- Probe: `ffprobe`
- Decode / transform: `ffmpeg` (analysis decode only; conversion pipeline uses explicit profiles)
- Tag writing: format-specific tools/libraries (chosen to support safe atomic writes and preserve tags)

## 5) Data model (minimum)
| Entity | Key fields |
|---|---|
| Library | id, name, root_path, scan_policy, read_only, output_paths, created_at |
| MediaFile | id, library_id, path, size, mtime, quick_hash, full_hash(optional), status |
| Track | id, media_file_id, duration, codec, sr, bit_depth, channels |
| ScanRun | id, library_id, started_at, finished_at, counts |
| AnalysisResult | id, track_id, version, lossless_score, issues_json, stats_json |
| Artifact | id, track_id, type, path, metadata_json |
| ActionLog | id, type(tag_edit/convert/etc), target_id, before_json, after_json, actor, created_at |
| ConversionJob | id, source_track/album refs, profile, status, progress, logs_path |

## 6) API surface (sketch)
| Method | Path | Purpose |
|---|---|
| GET | /api/libraries | list libraries |
| POST | /api/libraries | create library |
| POST | /api/libraries/{id}/scan | trigger scan now |
| GET | /api/tracks?filter=issues | triage view |
| GET | /api/tracks/{id} | details + results |
| GET | /api/tracks/{id}/artifacts | list evidence artifacts |
| POST | /api/tracks/{id}/tags | write metadata (diff required) |
| POST | /api/conversions | enqueue conversion |
| GET | /api/jobs | job status |

## 7) Non-goals (v1)
| Non-goal | Why |
|---|---|
| Perfect “proof” of original mastering/source | Detecting lossy ancestry is probabilistic; we provide evidence + confidence, not certainty |
| Automatic deletion of “bad” music | Too risky; only flag + optionally quarantine copies |
| DRM unlocking | Out of scope |

## 8) Security & access
| Area | Requirement |
|---|---|
| Auth | local login for MVP; optional OIDC later |
| Permissions | roles: admin/editor/viewer; read-only libraries enforce no writes |
| NAS mounts | recommend mounting read-only for scan-only libraries |
| SSH deploy | key-based, passwordless; restrict to a deploy user with least privilege |

## 9) Deployment (AlmaLinux, VM)
| Item | Approach |
|---|---|
| Build | produce a single Go binary + static assets |
| Run | systemd service (or container later); reverse proxy via Caddy/Nginx |
| Config | YAML/TOML config + DB DSN + artifact path |
| Observability | structured logs + basic metrics endpoint + job log viewer in UI |

## 10) Acceptance criteria (v1)
| Requirement | Pass condition |
|---|---|
| Incremental scan | Adding files to NAS results in jobs within configured interval, no full re-scan required |
| Evidence UI | Any “lossy suspicion” warning links to at least one chart + exportable JSON |
| Metadata editor | Can edit common tags safely and logs before/after |
| Conversion queue | Can enqueue and process conversions later, with progress visibility |
| Idempotency | Re-running scan does not duplicate DB rows or artifacts unnecessarily |
