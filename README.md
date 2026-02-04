<p align="center">
  <img src="web/static/img/logo.svg" alt="Ottavia Logo" width="120" height="120">
</p>

<h1 align="center">Ottavia</h1>
<h3 align="center">Music Quality Lab</h3>

<p align="center">
  A self-hosted tool for monitoring music libraries, verifying lossless audio integrity,<br>
  auditing metadata, and managing format conversions.
</p>

<p align="center">
  <a href="#features">Features</a> •
  <a href="#screenshots">Screenshots</a> •
  <a href="#installation">Installation</a> •
  <a href="#usage">Usage</a> •
  <a href="#api">API</a> •
  <a href="#contributing">Contributing</a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go Version">
  <img src="https://img.shields.io/badge/GOAT_Stack-Go%20%2B%20templ%20%2B%20Alpine%20%2B%20Tailwind-purple?style=flat-square" alt="GOAT Stack">
  <img src="https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/FFmpeg-007808?style=flat-square&logo=ffmpeg&logoColor=white" alt="FFmpeg">
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Platform-AlmaLinux%20%7C%20Linux%20%7C%20macOS-lightgrey?style=flat-square" alt="Platform">
  <img src="https://img.shields.io/badge/License-Blue_Oak-blue?style=flat-square" alt="License">
  <img src="https://img.shields.io/badge/PRs-Welcome-brightgreen?style=flat-square" alt="PRs Welcome">
</p>

<p align="center">
  <img src="https://img.shields.io/github/stars/ParkWardRR/Ottavia-NAS-Music-Quality-Lab?style=flat-square&logo=github&color=yellow" alt="Stars">
  <img src="https://img.shields.io/github/forks/ParkWardRR/Ottavia-NAS-Music-Quality-Lab?style=flat-square&logo=github" alt="Forks">
  <img src="https://img.shields.io/github/issues/ParkWardRR/Ottavia-NAS-Music-Quality-Lab?style=flat-square&logo=github" alt="Issues">
  <img src="https://img.shields.io/github/last-commit/ParkWardRR/Ottavia-NAS-Music-Quality-Lab?style=flat-square&logo=github" alt="Last Commit">
</p>

---

## ✨ Features

### 🔍 **Lossless Authenticity Verification**
- Detects files that may have been transcoded from lossy sources
- Analyzes high-frequency cutoff and spectral rolloff characteristics
- Provides confidence scores (0-100%) with plain-English explanations
- Visual "Verified" or "Suspicious" badges for quick assessment
- Evidence export for further analysis

### 📊 **Pro-Level Audio Analysis**
- Peak level and true peak detection (dBFS)
- Clipping detection with exact sample counts
- Loudness measurement (Integrated LUFS and Loudness Range)
- DC offset and phase correlation analysis
- Waveform visualization with peak markers
- Spectrogram artifact generation
- File integrity verification (detects truncated/corrupted streams)

### 📈 **Dynamic Range (DR) Scoring**
- Industry-standard DR measurement for "loudness war" detection
- Visual DR scale from "Crushed" to "Excellent"
- Color-coded badges (DR14+ = Excellent, DR8-13 = Good, etc.)
- Human-readable explanations for audiophiles and beginners
- Quick assessment sidebar with Pass/Fail indicators

### 🏷️ **Metadata Management**
- Comprehensive tag auditing (title, artist, album, year, genre, etc.)
- Missing/inconsistent tag detection
- Artwork presence and dimension analysis
- Tag display in track detail view
- Album-level quality badges

### 🔄 **Format Conversion**
- Built-in conversion profiles (iPod, Red Book compatible)
- Queue-based job processing with worker pool
- Background processing with progress tracking
- Scan interval configuration per library

### 🖥️ **Modern Apple-Inspired UI**
- Clean, professional glassmorphism design
- Dark/Light theme with system preference detection
- Responsive layout (desktop, tablet, mobile)
- Real-time updates with HTMX
- Beautiful gradient accents and smooth animations

### 🏠 **Self-Hosted & NAS-Friendly**
- Single binary deployment
- SQLite database (no external dependencies)
- Low resource footprint
- Works great on Synology, QNAP, Unraid, TrueNAS
- Tested on AlmaLinux, Ubuntu, macOS

---

## 📸 Screenshots

<p align="center">
  <img src="screenshots/dashboard-light.png" alt="Dashboard Light" width="100%">
  <br>
  <em>Dashboard - Light Mode</em>
</p>

<p align="center">
  <img src="screenshots/dashboard-dark.png" alt="Dashboard Dark" width="100%">
  <br>
  <em>Dashboard - Dark Mode</em>
</p>

<p align="center">
  <img src="screenshots/tracks-light.png" alt="Tracks" width="100%">
  <br>
  <em>Tracks Browser</em>
</p>

<p align="center">
  <img src="screenshots/track-detail-dark.png" alt="Track Analysis" width="100%">
  <br>
  <em>Pro-Level Track Analysis - Dark Mode</em>
</p>

<p align="center">
  <img src="screenshots/settings-dark.png" alt="Settings" width="100%">
  <br>
  <em>Settings - Dark Mode</em>
</p>

<p align="center">
  <img src="screenshots/dashboard-mobile.png" alt="Mobile" width="300">
  <br>
  <em>Mobile View</em>
</p>

---

## 🚀 Installation

### Prerequisites

- Go 1.22+
- Node.js 18+ (for Tailwind CSS)
- FFmpeg and FFprobe
- SQLite 3.35+

### Quick Start

```bash
# Clone the repository
git clone https://github.com/ParkWardRR/Ottavia-NAS-Music-Quality-Lab.git
cd ottavia

# Install dependencies and build
make deps
make all

# Run the server
make run
```

Open http://localhost:8080 in your browser.

### Using Go Install

```bash
go install github.com/ParkWardRR/Ottavia-NAS-Music-Quality-Lab/cmd/server@latest
```

### Docker

```bash
docker pull ghcr.io/ParkWardRR/Ottavia-NAS-Music-Quality-Lab:latest
docker run -p 8080:8080 -v /path/to/music:/music -v ottavia-data:/data ghcr.io/ParkWardRR/Ottavia-NAS-Music-Quality-Lab
```

### From Source

```bash
# Install templ
go install github.com/a-h/templ/cmd/templ@latest

# Install npm dependencies
npm install

# Build everything
make all

# Or step by step
make templ   # Generate templates
make css     # Build Tailwind CSS
make build   # Compile binary
```

---

## ⚙️ Configuration

Create a `config.yaml` file:

```yaml
server:
  host: "0.0.0.0"
  port: 8080

database:
  driver: "sqlite3"
  dsn: "./ottavia.db"

scanner:
  default_interval: "15m"
  worker_count: 4
  batch_size: 100
  max_retries: 3

storage:
  artifacts_path: "./artifacts/data"
  temp_path: "./artifacts/temp"

ffmpeg:
  ffprobe_path: "ffprobe"
  ffmpeg_path: "ffmpeg"
```

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OTTAVIA_CONFIG` | Config file path | `./config.yaml` |
| `OTTAVIA_PORT` | Server port | `8080` |
| `OTTAVIA_DB_DSN` | Database DSN | `./ottavia.db` |
| `OTTAVIA_DEBUG` | Enable debug mode | `false` |

---

## 📖 Usage

### Adding a Library

1. Click **Add Library** on the dashboard
2. Enter a name and the path to your music folder
3. Set scan interval and read-only mode
4. Click **Add Library**

### Running a Scan

Libraries are scanned automatically at the configured interval. To scan manually:

1. Go to the library card
2. Click the refresh icon
3. Monitor progress in the Jobs view

### Viewing Analysis Results

1. Navigate to **Tracks**
2. Click on any track to view details
3. Review analysis results and evidence
4. Export evidence as JSON/PNG

### Editing Metadata

1. Open track details
2. Click **Edit** in the Metadata section
3. Make changes and preview diff
4. Click **Save** to apply

### Converting Files

1. Select tracks or albums
2. Choose a conversion profile
3. Click **Convert**
4. Monitor progress in Jobs

---

## 🔌 API Reference

### Libraries

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/libraries` | List all libraries |
| `POST` | `/api/libraries` | Create a library |
| `GET` | `/api/libraries/:id` | Get library details |
| `PUT` | `/api/libraries/:id` | Update a library |
| `DELETE` | `/api/libraries/:id` | Delete a library |
| `POST` | `/api/libraries/:id/scan` | Trigger a scan |

### Tracks

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/tracks` | List tracks (with filters) |
| `GET` | `/api/tracks/:id` | Get track details |
| `POST` | `/api/tracks/:id/tags` | Update track tags |
| `GET` | `/api/tracks/:id/artifacts` | List evidence artifacts |

### Settings

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/settings` | Get all settings |
| `POST` | `/api/settings` | Update settings |

### Example Request

```bash
# Create a library
curl -X POST http://localhost:8080/api/libraries \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Music",
    "rootPath": "/mnt/nas/music",
    "scanInterval": "1h",
    "readOnly": true
  }'

# List tracks with issues
curl "http://localhost:8080/api/tracks?filter=issues&limit=20"
```

---

## 🧪 Testing

### Unit Tests

```bash
make test
```

### E2E Tests (Playwright)

```bash
# Start the server first
make run &

# Run E2E tests
make test-e2e
```

### Generate Screenshots

```bash
make screenshots
```

---

## 🛠️ Development

### Project Structure

```
ottavia/
├── cmd/
│   └── server/          # Main application entry
├── internal/
│   ├── analyzer/        # Audio analysis engine
│   ├── config/          # Configuration handling
│   ├── database/        # Database layer
│   ├── handlers/        # HTTP handlers
│   ├── models/          # Data models
│   ├── scanner/         # Library scanner
│   └── services/        # Business logic
├── migrations/          # Database migrations
├── tests/
│   └── e2e/            # Playwright tests
└── web/
    ├── static/          # CSS, JS, images
    └── templates/       # Templ templates
```

### Hot Reload Development

```bash
# Install air for hot reload
go install github.com/cosmtrek/air@latest

# Run with hot reload
make dev
```

### Code Style

```bash
# Format code
make fmt

# Run linters
make lint
```

---

## 🗺️ Roadmap

- [x] **Phase 0**: Project foundation + Ottavia branding
- [x] **Phase 1**: Scanner MVP with job queue
- [x] **Phase 2**: FFprobe integration + audio analysis
- [x] **Phase 3**: Lossy detection + DR scoring + album overhaul
- [ ] **Phase 4**: Metadata editor + bulk operations
- [ ] **Phase 5**: Conversion queue + progress UI
- [ ] **Phase 6**: Hardening + production deployment

### Recently Completed
- Pro-level track analysis page with quality badges
- Lossless authenticity verification with confidence scores
- Dynamic Range (DR) scoring with visual scale
- Quick Assessment sidebar with Pass/Fail indicators
- Screenshots with populated music data
- Nginx reverse proxy deployment on AlmaLinux

See [roadmap.md](roadmap.md) for detailed progress.

---

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the [Blue Oak Model License 1.0.0](LICENSE).

---

## 🙏 Acknowledgments

- [FFmpeg](https://ffmpeg.org/) - Audio analysis and conversion
- [templ](https://templ.guide/) - Go HTML templating
- [Tailwind CSS](https://tailwindcss.com/) - Styling
- [Alpine.js](https://alpinejs.dev/) - Interactivity
- [Playwright](https://playwright.dev/) - E2E testing

---

<p align="center">
  <strong>Made with ❤️ for audiophiles everywhere</strong>
</p>

<p align="center">
  <a href="https://github.com/ParkWardRR/Ottavia-NAS-Music-Quality-Lab">
    <img src="https://img.shields.io/badge/⭐_Star_this_repo-If_you_find_it_useful!-yellow?style=for-the-badge" alt="Star">
  </a>
</p>
