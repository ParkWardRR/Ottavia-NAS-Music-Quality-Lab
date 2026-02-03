-- Seville Music Quality Lab - Initial Schema

-- Libraries table
CREATE TABLE IF NOT EXISTS libraries (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    root_path TEXT NOT NULL UNIQUE,
    scan_interval TEXT NOT NULL DEFAULT '15m',
    read_only INTEGER NOT NULL DEFAULT 1,
    output_path TEXT,
    allowed_formats TEXT,
    last_scan_at DATETIME,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Media files table
CREATE TABLE IF NOT EXISTS media_files (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    filename TEXT NOT NULL,
    extension TEXT NOT NULL,
    size INTEGER NOT NULL,
    mtime DATETIME NOT NULL,
    quick_hash TEXT,
    full_hash TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    error_msg TEXT,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    UNIQUE(library_id, path)
);

CREATE INDEX IF NOT EXISTS idx_media_files_library ON media_files(library_id);
CREATE INDEX IF NOT EXISTS idx_media_files_status ON media_files(status);
CREATE INDEX IF NOT EXISTS idx_media_files_path ON media_files(path);

-- Tracks table
CREATE TABLE IF NOT EXISTS tracks (
    id TEXT PRIMARY KEY,
    media_file_id TEXT NOT NULL UNIQUE REFERENCES media_files(id) ON DELETE CASCADE,
    duration REAL NOT NULL,
    codec TEXT NOT NULL,
    sample_rate INTEGER NOT NULL,
    bit_depth INTEGER NOT NULL,
    channels INTEGER NOT NULL,
    bitrate INTEGER NOT NULL DEFAULT 0,
    title TEXT,
    artist TEXT,
    album TEXT,
    album_artist TEXT,
    track_number INTEGER,
    disc_number INTEGER,
    year INTEGER,
    genre TEXT,
    has_artwork INTEGER NOT NULL DEFAULT 0,
    artwork_width INTEGER,
    artwork_height INTEGER,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_tracks_album ON tracks(album);
CREATE INDEX IF NOT EXISTS idx_tracks_artist ON tracks(artist);

-- Scan runs table
CREATE TABLE IF NOT EXISTS scan_runs (
    id TEXT PRIMARY KEY,
    library_id TEXT NOT NULL REFERENCES libraries(id) ON DELETE CASCADE,
    status TEXT NOT NULL,
    files_found INTEGER NOT NULL DEFAULT 0,
    files_new INTEGER NOT NULL DEFAULT 0,
    files_changed INTEGER NOT NULL DEFAULT 0,
    files_deleted INTEGER NOT NULL DEFAULT 0,
    files_failed INTEGER NOT NULL DEFAULT 0,
    started_at DATETIME NOT NULL,
    finished_at DATETIME,
    error_msg TEXT
);

CREATE INDEX IF NOT EXISTS idx_scan_runs_library ON scan_runs(library_id);

-- Analysis results table
CREATE TABLE IF NOT EXISTS analysis_results (
    id TEXT PRIMARY KEY,
    track_id TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    version INTEGER NOT NULL DEFAULT 1,
    lossless_score REAL NOT NULL DEFAULT 0,
    lossless_status TEXT NOT NULL DEFAULT 'pending',
    integrity_ok INTEGER NOT NULL DEFAULT 1,
    decode_errors INTEGER NOT NULL DEFAULT 0,
    peak_level REAL NOT NULL DEFAULT 0,
    true_peak REAL NOT NULL DEFAULT 0,
    crest_factor REAL NOT NULL DEFAULT 0,
    clipped_samples INTEGER NOT NULL DEFAULT 0,
    dc_offset REAL NOT NULL DEFAULT 0,
    integrated_loudness REAL NOT NULL DEFAULT -23,
    loudness_range REAL NOT NULL DEFAULT 0,
    high_freq_cutoff REAL NOT NULL DEFAULT 22050,
    spectral_rolloff REAL NOT NULL DEFAULT 0,
    phase_correlation REAL NOT NULL DEFAULT 1,
    issues_json TEXT NOT NULL DEFAULT '[]',
    stats_json TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_analysis_results_track ON analysis_results(track_id);
CREATE INDEX IF NOT EXISTS idx_analysis_results_status ON analysis_results(lossless_status);

-- Artifacts table
CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    track_id TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    path TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    width INTEGER,
    height INTEGER,
    metadata_json TEXT,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_track ON artifacts(track_id);
CREATE INDEX IF NOT EXISTS idx_artifacts_type ON artifacts(type);

-- Action logs table
CREATE TABLE IF NOT EXISTS action_logs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    actor TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_action_logs_target ON action_logs(target_type, target_id);
CREATE INDEX IF NOT EXISTS idx_action_logs_created ON action_logs(created_at);

-- Jobs table
CREATE TABLE IF NOT EXISTS jobs (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    priority INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    payload_json TEXT,
    scheduled_at DATETIME NOT NULL,
    started_at DATETIME,
    finished_at DATETIME,
    created_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_type ON jobs(type);
CREATE INDEX IF NOT EXISTS idx_jobs_scheduled ON jobs(scheduled_at);

-- Conversion jobs table
CREATE TABLE IF NOT EXISTS conversion_jobs (
    id TEXT PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    profile TEXT NOT NULL,
    output_path TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    progress REAL NOT NULL DEFAULT 0,
    logs_path TEXT,
    error_msg TEXT,
    queued_at DATETIME NOT NULL,
    started_at DATETIME,
    finished_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_conversion_jobs_status ON conversion_jobs(status);

-- Conversion profiles table
CREATE TABLE IF NOT EXISTS conversion_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    codec TEXT NOT NULL,
    sample_rate INTEGER NOT NULL,
    bit_depth INTEGER NOT NULL DEFAULT 0,
    bitrate INTEGER NOT NULL DEFAULT 0,
    options TEXT NOT NULL DEFAULT '{}',
    is_builtin INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

-- Settings table
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'string',
    category TEXT NOT NULL DEFAULT 'general',
    updated_at DATETIME NOT NULL
);

-- Album view for consistency checks
CREATE VIEW IF NOT EXISTS album_tracks AS
SELECT
    t.album,
    t.album_artist,
    t.year,
    m.library_id,
    COUNT(*) as track_count,
    SUM(t.duration) as total_duration,
    GROUP_CONCAT(DISTINCT t.genre) as genres
FROM tracks t
JOIN media_files m ON t.media_file_id = m.id
WHERE t.album IS NOT NULL
GROUP BY t.album, t.album_artist, m.library_id;
