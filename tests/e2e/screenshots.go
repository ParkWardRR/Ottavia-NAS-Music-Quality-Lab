//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	// Start the server in background
	serverCmd := exec.Command("./bin/ottavia", "-debug")
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	if err := serverCmd.Start(); err != nil {
		log.Printf("Note: Server may already be running or binary not built")
	}
	defer func() {
		if serverCmd.Process != nil {
			serverCmd.Process.Kill()
		}
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

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

	// Screenshots to capture
	screenshots := []struct {
		name     string
		path     string
		width    int
		height   int
		darkMode bool
		wait     time.Duration
	}{
		{"dashboard-light", "/", 1920, 1080, false, 500 * time.Millisecond},
		{"dashboard-dark", "/", 1920, 1080, true, 500 * time.Millisecond},
		{"tracks-light", "/tracks", 1920, 1080, false, 500 * time.Millisecond},
		{"tracks-dark", "/tracks", 1920, 1080, true, 500 * time.Millisecond},
		{"settings-light", "/settings", 1920, 1080, false, 500 * time.Millisecond},
		{"settings-dark", "/settings", 1920, 1080, true, 500 * time.Millisecond},
		{"dashboard-mobile", "/", 375, 812, false, 500 * time.Millisecond},
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
		_, err = page.Goto(fmt.Sprintf("http://localhost:8080%s", s.path), playwright.PageGotoOptions{
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
			FullPage: playwright.Bool(false),
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
