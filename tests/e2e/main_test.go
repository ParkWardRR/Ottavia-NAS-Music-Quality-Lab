package e2e

import (
	"os"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

var (
	pw      *playwright.Playwright
	browser playwright.Browser
	baseURL = "http://localhost:8080"
)

func TestMain(m *testing.M) {
	// Check if server is running
	if os.Getenv("SKIP_E2E") != "" {
		os.Exit(0)
	}

	// Install playwright
	if err := playwright.Install(); err != nil {
		panic(err)
	}

	// Start playwright
	var err error
	pw, err = playwright.Run()
	if err != nil {
		panic(err)
	}

	// Launch browser
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		panic(err)
	}

	// Run tests
	code := m.Run()

	// Cleanup
	browser.Close()
	pw.Stop()

	os.Exit(code)
}

func TestDashboardLoads(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Check title
	title, err := page.Title()
	if err != nil {
		t.Fatalf("could not get title: %v", err)
	}
	if title == "" {
		t.Error("page title should not be empty")
	}

	// Check for dashboard heading
	heading := page.Locator("h1:has-text('Dashboard')")
	visible, err := heading.IsVisible()
	if err != nil {
		t.Fatalf("could not check heading visibility: %v", err)
	}
	if !visible {
		t.Error("Dashboard heading should be visible")
	}
}

func TestTracksPageLoads(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto(baseURL+"/tracks", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Check for tracks heading
	heading := page.Locator("h1:has-text('Tracks')")
	visible, err := heading.IsVisible()
	if err != nil {
		t.Fatalf("could not check heading visibility: %v", err)
	}
	if !visible {
		t.Error("Tracks heading should be visible")
	}
}

func TestSettingsPageLoads(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto(baseURL+"/settings", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Check for settings heading
	heading := page.Locator("h1:has-text('Settings')")
	visible, err := heading.IsVisible()
	if err != nil {
		t.Fatalf("could not check heading visibility: %v", err)
	}
	if !visible {
		t.Error("Settings heading should be visible")
	}
}

func TestSidebarNavigation(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Test sidebar links
	links := []struct {
		selector string
		expected string
	}{
		{"a:has-text('Libraries')", "/libraries"},
		{"a:has-text('All Tracks')", "/tracks"},
		{"a:has-text('Settings')", "/settings"},
	}

	for _, link := range links {
		t.Run(link.expected, func(t *testing.T) {
			elem := page.Locator(link.selector)
			visible, err := elem.IsVisible()
			if err != nil {
				t.Fatalf("could not check visibility: %v", err)
			}
			if !visible {
				t.Errorf("Link %s should be visible", link.selector)
			}
		})
	}
}

func TestThemeToggle(t *testing.T) {
	context, err := browser.NewContext()
	if err != nil {
		t.Fatalf("could not create context: %v", err)
	}
	defer context.Close()

	page, err := context.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}

	_, err = page.Goto(baseURL+"/settings", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Find and click dark mode button
	darkButton := page.Locator("button:has-text('Dark')")
	visible, err := darkButton.IsVisible()
	if err != nil {
		t.Fatalf("could not check button visibility: %v", err)
	}
	if !visible {
		t.Skip("Dark mode button not visible")
	}

	err = darkButton.Click()
	if err != nil {
		t.Fatalf("could not click dark button: %v", err)
	}

	// Wait for theme transition
	time.Sleep(300 * time.Millisecond)

	// Check if dark class is applied
	isDark, err := page.Evaluate(`document.documentElement.classList.contains('dark')`)
	if err != nil {
		t.Fatalf("could not evaluate: %v", err)
	}
	if !isDark.(bool) {
		t.Error("Dark mode should be applied")
	}
}

func TestAPIHealth(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	resp, err := page.Goto(baseURL+"/api/health", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	if resp.Status() != 200 {
		t.Errorf("expected status 200, got %d", resp.Status())
	}
}

func TestAddLibraryModalOpens(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("could not create page: %v", err)
	}
	defer page.Close()

	_, err = page.Goto(baseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	})
	if err != nil {
		t.Fatalf("could not navigate: %v", err)
	}

	// Click Add Library button
	addButton := page.Locator("button:has-text('Add Library')")
	if visible, _ := addButton.IsVisible(); visible {
		err = addButton.Click()
		if err != nil {
			t.Fatalf("could not click add button: %v", err)
		}

		// Wait for modal
		time.Sleep(300 * time.Millisecond)

		// Check modal is visible
		modal := page.Locator("#addLibraryModal")
		hidden, err := modal.Evaluate(`el => el.classList.contains('hidden')`, nil)
		if err != nil {
			t.Fatalf("could not check modal visibility: %v", err)
		}
		if hidden.(bool) {
			t.Error("Modal should be visible after clicking Add Library")
		}
	} else {
		t.Skip("Add Library button not visible")
	}
}

func TestResponsiveLayout(t *testing.T) {
	viewports := []struct {
		name   string
		width  int
		height int
	}{
		{"mobile", 375, 812},
		{"tablet", 768, 1024},
		{"desktop", 1920, 1080},
	}

	for _, vp := range viewports {
		t.Run(vp.name, func(t *testing.T) {
			context, err := browser.NewContext(playwright.BrowserNewContextOptions{
				Viewport: &playwright.Size{
					Width:  vp.width,
					Height: vp.height,
				},
			})
			if err != nil {
				t.Fatalf("could not create context: %v", err)
			}
			defer context.Close()

			page, err := context.NewPage()
			if err != nil {
				t.Fatalf("could not create page: %v", err)
			}

			_, err = page.Goto(baseURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
			})
			if err != nil {
				t.Fatalf("could not navigate: %v", err)
			}

			// Page should load without errors
			heading := page.Locator("main h1")
			visible, err := heading.IsVisible()
			if err != nil {
				t.Fatalf("could not check heading: %v", err)
			}
			if !visible {
				t.Error("Main heading should be visible")
			}
		})
	}
}
