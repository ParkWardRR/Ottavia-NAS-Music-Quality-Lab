package scanner

import (
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/ottavia-music/ottavia/internal/database"
	"github.com/ottavia-music/ottavia/internal/models"
)

var supportedExtensions = map[string]bool{
	".flac": true,
	".alac": true,
	".wav":  true,
	".aiff": true,
	".aif":  true,
	".mp3":  true,
	".m4a":  true,
	".aac":  true,
	".ogg":  true,
	".opus": true,
	".wma":  true,
	".ape":  true,
	".wv":   true,
	".dsf":  true,
	".dff":  true,
}

type Scanner struct {
	db          *database.DB
	workerCount int
	batchSize   int

	running   bool
	runningMu sync.Mutex
	cancel    context.CancelFunc
}

func New(db *database.DB, workerCount, batchSize int) *Scanner {
	return &Scanner{
		db:          db,
		workerCount: workerCount,
		batchSize:   batchSize,
	}
}

type ScanResult struct {
	Run     *models.ScanRun
	NewJobs []string
	Errors  []error
}

func (s *Scanner) ScanLibrary(ctx context.Context, libraryID string) (*ScanResult, error) {
	s.runningMu.Lock()
	if s.running {
		s.runningMu.Unlock()
		return nil, fmt.Errorf("scan already in progress")
	}
	s.running = true
	ctx, s.cancel = context.WithCancel(ctx)
	s.runningMu.Unlock()

	defer func() {
		s.runningMu.Lock()
		s.running = false
		s.runningMu.Unlock()
	}()

	lib, err := s.db.GetLibrary(ctx, libraryID)
	if err != nil {
		return nil, fmt.Errorf("failed to get library: %w", err)
	}

	run := &models.ScanRun{
		LibraryID: libraryID,
	}
	if err := s.db.CreateScanRun(ctx, run); err != nil {
		return nil, fmt.Errorf("failed to create scan run: %w", err)
	}

	result := &ScanResult{Run: run}

	log.Info().
		Str("library_id", libraryID).
		Str("root_path", lib.RootPath).
		Msg("Starting library scan")

	existingFiles := make(map[string]*models.MediaFile)
	files, err := s.db.ListMediaFiles(ctx, libraryID)
	if err != nil && err != sql.ErrNoRows {
		result.Errors = append(result.Errors, err)
	}
	for i := range files {
		existingFiles[files[i].Path] = &files[i]
	}

	foundPaths := make(map[string]bool)
	var scanErrors []error

	err = filepath.WalkDir(lib.RootPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("walk error at %s: %w", path, err))
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExtensions[ext] {
			return nil
		}

		run.FilesFound++
		foundPaths[path] = true

		info, err := d.Info()
		if err != nil {
			scanErrors = append(scanErrors, fmt.Errorf("stat error at %s: %w", path, err))
			return nil
		}

		existing, exists := existingFiles[path]
		if exists {
			if existing.Size == info.Size() && existing.Mtime.Unix() == info.ModTime().Unix() {
				return nil
			}
			run.FilesChanged++
			existing.Size = info.Size()
			existing.Mtime = info.ModTime()
			existing.Status = models.StatusPending
			existing.QuickHash = sql.NullString{}
			if err := s.db.UpdateMediaFile(ctx, existing); err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("update error at %s: %w", path, err))
			}
		} else {
			run.FilesNew++
			mf := &models.MediaFile{
				LibraryID: libraryID,
				Path:      path,
				Filename:  filepath.Base(path),
				Extension: ext,
				Size:      info.Size(),
				Mtime:     info.ModTime(),
			}
			if err := s.db.CreateMediaFile(ctx, mf); err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("create error at %s: %w", path, err))
			}

			job := &models.Job{
				Type:        "analyze",
				TargetType:  "media_file",
				TargetID:    mf.ID,
				Priority:    0,
				MaxAttempts: 3,
				ScheduledAt: time.Now(),
			}
			if err := s.db.CreateJob(ctx, job); err != nil {
				scanErrors = append(scanErrors, fmt.Errorf("job create error: %w", err))
			} else {
				result.NewJobs = append(result.NewJobs, job.ID)
			}
		}

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, err)
	}
	result.Errors = append(result.Errors, scanErrors...)

	for path, mf := range existingFiles {
		if !foundPaths[path] {
			run.FilesDeleted++
			mf.Status = "deleted"
			s.db.UpdateMediaFile(ctx, mf)
		}
	}

	run.FilesFailed = len(result.Errors)
	run.Status = models.StatusSuccess
	if len(result.Errors) > 0 {
		run.Status = models.StatusFailed
		run.ErrorMsg = sql.NullString{String: result.Errors[0].Error(), Valid: true}
	}
	run.FinishedAt = sql.NullTime{Time: time.Now(), Valid: true}

	if err := s.db.UpdateScanRun(ctx, run); err != nil {
		result.Errors = append(result.Errors, err)
	}

	lib.LastScanAt = sql.NullTime{Time: time.Now(), Valid: true}
	lib.Status = models.StatusSuccess
	if err := s.db.UpdateLibrary(ctx, lib); err != nil {
		result.Errors = append(result.Errors, err)
	}

	log.Info().
		Str("library_id", libraryID).
		Int("found", run.FilesFound).
		Int("new", run.FilesNew).
		Int("changed", run.FilesChanged).
		Int("deleted", run.FilesDeleted).
		Msg("Scan completed")

	return result, nil
}

func (s *Scanner) Stop() {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Scanner) IsRunning() bool {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()
	return s.running
}

func quickHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := md5.New()
	buf := make([]byte, 64*1024)

	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	h.Write(buf[:n])

	info, _ := f.Stat()
	if info.Size() > 64*1024 {
		f.Seek(-64*1024, io.SeekEnd)
		n, err = f.Read(buf)
		if err != nil && err != io.EOF {
			return "", err
		}
		h.Write(buf[:n])
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func generateID() string {
	return uuid.NewString()
}
