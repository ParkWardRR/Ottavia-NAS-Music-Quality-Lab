//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	// Get base URL from environment or default
	baseURL := os.Getenv("OTTAVIA_URL")
	if baseURL == "" {
		baseURL = "http://ottavia" // Default to remote server
	}

	// Install playwright browsers
	if err := playwright.Install(); err != nil {
		log.Fatalf("could not install playwright: %v", err)
	}

	// Start playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("could not start playwright: %v", err)
	}
	defer pw.Stop()

	// Launch browser
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		log.Fatalf("could not launch browser: %v", err)
	}
	defer browser.Close()

	// Create screenshots directory
	screenshotsDir := "screenshots"
	if err := os.MkdirAll(screenshotsDir, 0755); err != nil {
		log.Fatalf("could not create screenshots dir: %v", err)
	}

	// Get a track ID for the track detail page
	trackID := os.Getenv("OTTAVIA_TRACK_ID")
	if trackID == "" {
		trackID = "6b6e7b79-eb0d-457e-b890-a72e7dd8bdd3" // Default track ID
	}

	// Get an album name for the album detail page
	albumName := os.Getenv("OTTAVIA_ALBUM_NAME")
	albumArtist := os.Getenv("OTTAVIA_ALBUM_ARTIST")
	if albumName == "" {
		albumName = "Unknown Album"
	}

	// Build album URL
	albumURL := fmt.Sprintf("/albums/%s", albumName)
	if albumArtist != "" {
		albumURL = fmt.Sprintf("/albums/%s?artist=%s", albumName, albumArtist)
	}

	// Screenshots to capture
	screenshots := []struct {
		name       string
		path       string
		width      int
		height     int
		darkMode   bool
		wait       time.Duration
		fullPage   bool
		showWizard bool
	}{
		{"dashboard-light", "/", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"dashboard-dark", "/", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"tracks-light", "/tracks", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"tracks-dark", "/tracks", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"track-detail-light", "/tracks/" + trackID, 1920, 1400, false, 500 * time.Millisecond, true, false},
		{"track-detail-dark", "/tracks/" + trackID, 1920, 1400, true, 500 * time.Millisecond, true, false},
		{"albums-light", "/albums", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"albums-dark", "/albums", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"album-detail-light", albumURL, 1920, 1400, false, 500 * time.Millisecond, true, false},
		{"album-detail-dark", albumURL, 1920, 1400, true, 500 * time.Millisecond, true, false},
		{"settings-light", "/settings", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"settings-dark", "/settings", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"conversions-light", "/conversions", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"conversions-dark", "/conversions", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"artwork-light", "/artwork", 1920, 1080, false, 500 * time.Millisecond, false, false},
		{"artwork-dark", "/artwork", 1920, 1080, true, 500 * time.Millisecond, false, false},
		{"wizard-welcome", "/", 1920, 1080, true, 800 * time.Millisecond, false, true},
		{"dashboard-mobile", "/", 390, 844, false, 500 * time.Millisecond, false, false},
		{"tracks-mobile", "/tracks", 390, 844, false, 500 * time.Millisecond, false, false},
	}

	for _, s := range screenshots {
		log.Printf("Capturing: %s", s.name)

		// Create new context with viewport
		context, err := browser.NewContext(playwright.BrowserNewContextOptions{
			Viewport: &playwright.Size{
				Width:  s.width,
				Height: s.height,
			},
			DeviceScaleFactor: playwright.Float(2), // Retina
			ColorScheme: func() *playwright.ColorScheme {
				if s.darkMode {
					return playwright.ColorSchemeDark
				}
				return playwright.ColorSchemeLight
			}(),
		})
		if err != nil {
			log.Printf("Error creating context for %s: %v", s.name, err)
			continue
		}

		page, err := context.NewPage()
		if err != nil {
			log.Printf("Error creating page for %s: %v", s.name, err)
			context.Close()
			continue
		}

		// Set localStorage to prevent wizard auto-show BEFORE navigating (except for wizard screenshot)
		if !s.showWizard {
			// First navigate to base URL to set localStorage
			_, err = page.Goto(baseURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateCommit,
			})
			if err == nil {
				page.Evaluate(`
					localStorage.setItem('welcomeWizardDismissed', 'true');
					localStorage.setItem('wizardsCompleted', JSON.stringify(['welcome']));
				`)
			}
		}

		// Navigate to target page
		_, err = page.Goto(fmt.Sprintf("%s%s", baseURL, s.path), playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		})
		if err != nil {
			log.Printf("Error navigating to %s: %v", s.path, err)
			context.Close()
			continue
		}

		// Wait for any animations to complete
		time.Sleep(s.wait)

		// Set dark mode if needed via JavaScript
		if s.darkMode {
			_, err = page.Evaluate(`document.documentElement.classList.add('dark')`)
			if err != nil {
				log.Printf("Error setting dark mode: %v", err)
			}
			time.Sleep(300 * time.Millisecond) // Wait for theme transition
		}

		// Handle wizard overlay
		if s.showWizard {
			// Show the welcome wizard - wait for Alpine first
			time.Sleep(500 * time.Millisecond)
			_, err = page.Evaluate(`
				if (typeof Alpine !== 'undefined' && Alpine.store('wizard')) {
					Alpine.store('wizard').start('welcome');
				}
			`)
			if err != nil {
				log.Printf("Error showing wizard: %v", err)
			}
			time.Sleep(500 * time.Millisecond) // Wait for wizard animation
		}

		// Take screenshot
		screenshotPath := filepath.Join(screenshotsDir, fmt.Sprintf("%s.png", s.name))
		_, err = page.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(screenshotPath),
			FullPage: playwright.Bool(s.fullPage),
		})
		if err != nil {
			log.Printf("Error taking screenshot %s: %v", s.name, err)
		} else {
			log.Printf("Saved: %s", screenshotPath)
		}

		context.Close()
	}

	log.Println("Screenshots complete!")
}
