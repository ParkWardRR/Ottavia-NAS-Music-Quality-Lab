package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/analyzer"
	"github.com/ottavia-music/ottavia/internal/config"
	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/handlers"
	"github.com/ottavia-music/ottavia/internal/jobs"
	"github.com/ottavia-music/ottavia/internal/metadata"
	"github.com/ottavia-music/ottavia/internal/models"
	"github.com/ottavia-music/ottavia/internal/scanner"
	"github.com/ottavia-music/ottavia/web/templates/pages"
)

var (
	version   = "1.0.0"
	buildTime = "development"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to config file")
	debug := flag.Bool("debug", false, "Enable debug logging")
	flag.Parse()

	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().
		Str("version", version).
		Str("build_time", buildTime).
		Msg("Starting Ottavia - Music Quality Lab")

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	// Initialize database
	db, err := database.New(cfg.Database.DSN)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Run migrations
	if err := db.Migrate(); err != nil {
		log.Fatal().Err(err).Msg("Failed to run migrations")
	}

	// Ensure directories exist
	for _, dir := range []string{cfg.Storage.ArtifactsPath, cfg.Storage.TempPath} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatal().Err(err).Str("dir", dir).Msg("Failed to create directory")
		}
	}

	// Initialize services
	scannerSvc := scanner.New(db, cfg.Scanner.WorkerCount, cfg.Scanner.BatchSize)
	analyzerSvc := analyzer.New(db, cfg.FFmpeg.FFprobePath, cfg.FFmpeg.FFmpegPath, cfg.Storage.ArtifactsPath)
	metadataWriter := metadata.New(db, cfg.FFmpeg.FFmpegPath)

	// Initialize handlers
	h := handlers.New(db, scannerSvc, analyzerSvc, metadataWriter)

	// Start job workers
	worker := jobs.NewWorker(db, analyzerSvc, cfg.Scanner.WorkerCount)
	worker.Start(context.Background())
	defer worker.Stop()

	// Start scheduler for periodic scans
	scheduler := jobs.NewScheduler(db, func(ctx context.Context, libraryID string) {
		result, err := scannerSvc.ScanLibrary(ctx, libraryID)
		if err != nil {
			log.Error().Err(err).Str("library_id", libraryID).Msg("Scheduled scan failed")
			return
		}
		log.Info().
			Str("library_id", libraryID).
			Int("new", result.Run.FilesNew).
			Int("changed", result.Run.FilesChanged).
			Msg("Scheduled scan completed")
	})
	scheduler.Start(context.Background())
	defer scheduler.Stop()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Static files - serve from filesystem for easier development
	execPath, _ := os.Executable()
	execDir := filepath.Dir(execPath)
	staticPath := filepath.Join(execDir, "..", "..", "web", "static")
	if _, err := os.Stat(staticPath); os.IsNotExist(err) {
		// Try current working directory
		staticPath = "web/static"
	}
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticPath))))

	// API routes
	r.Route("/api", func(r chi.Router) {
		// Health
		r.Get("/health", h.HealthCheck)
		r.Get("/stats", h.GetDashboardStats)

		// Libraries
		r.Get("/libraries", h.ListLibraries)
		r.Post("/libraries", h.CreateLibrary)
		r.Get("/libraries/{id}", h.GetLibrary)
		r.Put("/libraries/{id}", h.UpdateLibrary)
		r.Delete("/libraries/{id}", h.DeleteLibrary)
		r.Post("/libraries/{id}/scan", h.ScanLibrary)
		r.Get("/libraries/{id}/scans", h.ListScanRuns)

		// Tracks
		r.Get("/tracks", h.ListTracks)
		r.Get("/tracks/{id}", h.GetTrack)
		r.Post("/tracks/{id}/tags", h.UpdateTrackTags)
		r.Post("/tracks/{id}/tags/preview", h.PreviewTrackTags)
		r.Get("/tracks/{id}/artifacts", h.GetTrackArtifacts)

		// Bulk metadata operations
		r.Post("/tracks/bulk/preview", h.PreviewBulkOperation)
		r.Post("/tracks/bulk/apply", h.ApplyBulkOperation)
		r.Post("/albums/normalize-artist", h.NormalizeAlbumArtist)
		r.Post("/albums/fix-numbering", h.FixTrackNumbering)

		// Jobs
		r.Get("/jobs", h.ListJobs)

		// Settings
		r.Get("/settings", h.GetSettings)
		r.Post("/settings", h.UpdateSettings)

		// Conversion profiles
		r.Get("/profiles", h.ListConversionProfiles)

		// Action logs
		r.Get("/logs", h.ListActionLogs)
	})

	// Page routes
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		stats, err := db.GetDashboardStats(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get dashboard stats")
			stats = &database.DashboardStats{}
		}
		libraries, err := db.ListLibraries(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list libraries")
			libraries = []models.Library{}
		}
		tracks, _, err := db.ListTracks(ctx, "", "", 10, 0)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list tracks")
			tracks = []models.Track{}
		}
		settings, _ := db.GetAllSettings(ctx)

		pages.Dashboard(stats, libraries, tracks, settings).Render(ctx, w)
	})

	r.Get("/libraries", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		libraries, _ := db.ListLibraries(ctx)
		settings, _ := db.GetAllSettings(ctx)

		// Render libraries page
		_ = libraries
		_ = settings
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	r.Get("/tracks", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		filter := r.URL.Query().Get("filter")
		tracks, total, _ := db.ListTracks(ctx, "", filter, 50, 0)
		settings, _ := db.GetAllSettings(ctx)

		pages.TracksPage(tracks, total, filter, settings).Render(ctx, w)
	})

	r.Get("/albums", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		albums, total, err := db.ListAlbums(ctx, 50, 0)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list albums")
			albums = []database.Album{}
		}
		settings, _ := db.GetAllSettings(ctx)

		pages.AlbumsPage(albums, total, settings).Render(ctx, w)
	})

	r.Get("/albums/{name}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		albumName := chi.URLParam(r, "name")
		artist := r.URL.Query().Get("artist")

		album, err := db.GetAlbumDetail(ctx, albumName, artist)
		if err != nil {
			log.Error().Err(err).Str("album", albumName).Msg("Failed to get album detail")
			http.Error(w, "Album not found", http.StatusNotFound)
			return
		}
		settings, _ := db.GetAllSettings(ctx)

		pages.AlbumDetailPage(album, settings).Render(ctx, w)
	})

	r.Get("/tracks/{id}", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		id := chi.URLParam(r, "id")
		track, err := db.GetTrack(ctx, id)
		if err != nil {
			http.Error(w, "Track not found", http.StatusNotFound)
			return
		}
		analysis, _ := db.GetAnalysisResult(ctx, id)
		artifacts, _ := db.ListArtifacts(ctx, id)
		settings, _ := db.GetAllSettings(ctx)

		pages.TrackDetail(track, analysis, artifacts, settings).Render(ctx, w)
	})

	r.Get("/settings", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		settings, _ := db.GetAllSettings(ctx)
		profiles, _ := db.ListConversionProfiles(ctx)

		pages.Settings(settings, profiles).Render(ctx, w)
	})

	// Catch-all for other pages
	r.Get("/issues", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tracks?filter=issues", http.StatusTemporaryRedirect)
	})

	r.Get("/evidence", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tracks", http.StatusTemporaryRedirect)
	})

	r.Get("/conversions", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		jobs, err := db.ListConversionJobs(ctx, 50)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list conversion jobs")
			jobs = []models.ConversionJob{}
		}
		profiles, err := db.ListConversionProfiles(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to list conversion profiles")
			profiles = []models.ConversionProfile{}
		}
		settings, _ := db.GetAllSettings(ctx)

		pages.ConversionsPage(jobs, profiles, settings).Render(ctx, w)
	})

	r.Get("/jobs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	r.Get("/duplicates", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/tracks", http.StatusTemporaryRedirect)
	})

	// Artifact file server
	artifactsFS := http.FileServer(http.Dir(cfg.Storage.ArtifactsPath))
	r.Handle("/artifacts/*", http.StripPrefix("/artifacts/", artifactsFS))

	// Create server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info().Str("addr", addr).Msg("Server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server stopped")
}
