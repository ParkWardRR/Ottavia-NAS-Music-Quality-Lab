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

	// Screenshots to capture
	screenshots := []struct {
		name     string
		path     string
		width    int
		height   int
		darkMode bool
		wait     time.Duration
		fullPage bool
	}{
		{"dashboard-light", "/", 1920, 1080, false, 500 * time.Millisecond, false},
		{"dashboard-dark", "/", 1920, 1080, true, 500 * time.Millisecond, false},
		{"tracks-light", "/tracks", 1920, 1080, false, 500 * time.Millisecond, false},
		{"tracks-dark", "/tracks", 1920, 1080, true, 500 * time.Millisecond, false},
		{"track-detail-light", "/tracks/" + trackID, 1920, 1400, false, 500 * time.Millisecond, true},
		{"track-detail-dark", "/tracks/" + trackID, 1920, 1400, true, 500 * time.Millisecond, true},
		{"settings-light", "/settings", 1920, 1080, false, 500 * time.Millisecond, false},
		{"settings-dark", "/settings", 1920, 1080, true, 500 * time.Millisecond, false},
		{"dashboard-mobile", "/", 375, 812, false, 500 * time.Millisecond, false},
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

		// Navigate to page
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
