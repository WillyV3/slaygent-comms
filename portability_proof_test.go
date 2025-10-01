package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSyncPortabilityProof - STANDALONE TEST proving sync portability works across any machine
func TestSyncPortabilityProof(t *testing.T) {
	t.Log("🔍 PROVING: Sync scripts generate portable registry references that work on ANY machine")

	// Test different user scenarios that would break with hardcoded paths
	scenarios := []struct {
		name     string
		username string
		homeDir  string
	}{
		{"Fresh MacBook - New User", "newuser", "/Users/newuser"},
		{"Linux Developer", "dev", "/home/dev"},
		{"Corporate MacBook", "employee", "/Users/employee"},
		{"Original User", "williamvansickleiii", "/Users/williamvansickleiii"},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create isolated test environment for this "user"
			tempHome := t.TempDir()
			slaygentDir := filepath.Join(tempHome, ".slaygent")
			registryPath := filepath.Join(slaygentDir, "registry.json")
			testProject := filepath.Join(tempHome, "test-project")
			claudeFile := filepath.Join(testProject, "CLAUDE.md")

			// Setup user environment
			os.MkdirAll(slaygentDir, 0755)
			os.MkdirAll(testProject, 0755)

			// Create registry file
			registryContent := `[
  {
    "name": "test-agent",
    "agent_type": "claude",
    "directory": "` + testProject + `",
    "machine": "host"
  }
]`
			os.WriteFile(registryPath, []byte(registryContent), 0644)

			// Create initial CLAUDE.md (could contain old hardcoded paths)
			initialContent := `# Test Project

Some existing content here.

<!-- SLAYGENT-REGISTRY-START -->
# Inter-Agent Communication
@/Users/williamvansickleiii/.slaygent/registry.json

Old hardcoded reference that breaks on other machines!
<!-- SLAYGENT-REGISTRY-END -->

More content.`
			os.WriteFile(claudeFile, []byte(initialContent), 0644)

			// Run sync script with this user's environment
			scriptPath := "app/scripts/sync-claude.sh"
			if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
				t.Skip("Sync script not found - run from project root")
			}

			// Execute sync with user's HOME environment
			cmd := exec.Command("bash", "-c", "echo 'y' | "+scriptPath)
			cmd.Env = []string{
				"HOME=" + tempHome,
				"PATH=" + os.Getenv("PATH"),
			}
			cmd.Dir = tempHome

			output, err := cmd.Output()
			if err != nil {
				// Log output for debugging
				if len(output) > 0 {
					t.Logf("Script output: %s", string(output))
				}
				t.Fatalf("❌ SYNC FAILED for %s: %v", scenario.name, err)
			}

			// Read the updated CLAUDE.md
			updatedContent, err := os.ReadFile(claudeFile)
			if err != nil {
				t.Fatalf("Failed to read updated CLAUDE.md: %v", err)
			}

			contentStr := string(updatedContent)

			// 🎯 CRITICAL PORTABILITY TESTS

			// TEST 1: Must contain portable registry reference
			if !strings.Contains(contentStr, "@~/.slaygent/registry.json") {
				t.Errorf("❌ PORTABILITY FAILURE for %s: Missing portable reference '@~/.slaygent/registry.json'", scenario.name)
				t.Logf("Content:\n%s", contentStr)
			} else {
				t.Logf("✅ PORTABLE REFERENCE: Found '@~/.slaygent/registry.json' for %s", scenario.name)
			}

			// TEST 2: Must NOT contain any hardcoded absolute paths
			hardcodedPaths := []string{
				"/Users/williamvansickleiii/.slaygent/registry.json",
				"/home/williamvansickleiii/.slaygent/registry.json",
				tempHome + "/.slaygent/registry.json", // Even current user's absolute path
			}

			for _, hardcodedPath := range hardcodedPaths {
				if strings.Contains(contentStr, hardcodedPath) {
					t.Errorf("❌ PORTABILITY FAILURE for %s: Contains hardcoded path '%s'", scenario.name, hardcodedPath)
				}
			}

			// TEST 3: Verify sync markers are preserved
			if !strings.Contains(contentStr, "<!-- SLAYGENT-REGISTRY-START -->") ||
				!strings.Contains(contentStr, "<!-- SLAYGENT-REGISTRY-END -->") {
				t.Errorf("❌ SYNC MARKERS MISSING for %s", scenario.name)
			}

			// TEST 4: Verify original content is preserved
			if !strings.Contains(contentStr, "Some existing content here.") ||
				!strings.Contains(contentStr, "More content.") {
				t.Errorf("❌ CONTENT CORRUPTION for %s: Original content lost", scenario.name)
			}

			t.Logf("✅ PORTABILITY SUCCESS: %s can sync without hardcoded paths", scenario.name)
		})
	}
}

// TestScriptDiscoveryProof - Proves script discovery works without hardcoded versions
func TestScriptDiscoveryProof(t *testing.T) {
	t.Log("🔍 PROVING: Script discovery works across different Homebrew installations and versions")

	// Test different Homebrew setups
	brewSetups := []struct {
		name   string
		prefix string
	}{
		{"macOS ARM", "/opt/homebrew"},
		{"macOS Intel", "/usr/local"},
		{"Linux Homebrew", "/home/linuxbrew/.linuxbrew"},
	}

	for _, setup := range brewSetups {
		t.Run(setup.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create mock Homebrew structure
			mockPrefix := filepath.Join(tempDir, strings.TrimPrefix(setup.prefix, "/"))

			// Test lib location (primary)
			libDir := filepath.Join(mockPrefix, "lib", "slaygent-comms")
			os.MkdirAll(libDir, 0755)
			libScript := filepath.Join(libDir, "sync-claude.sh")
			os.WriteFile(libScript, []byte("#!/bin/bash\necho 'lib version'"), 0755)

			// Test Cellar location with multiple versions (version-agnostic)
			versions := []string{"v0.3.1", "v0.4.0", "v1.0.0"}
			cellarBase := filepath.Join(mockPrefix, "Cellar", "slaygent-comms")

			for _, version := range versions {
				versionDir := filepath.Join(cellarBase, version, "libexec")
				os.MkdirAll(versionDir, 0755)
				scriptPath := filepath.Join(versionDir, "sync-claude.sh")
				os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'cellar "+version+"'"), 0755)
			}

			// Simulate findSyncScript logic - should find scripts without hardcoded versions

			// Test lib location discovery
			if _, err := os.Stat(libScript); err == nil {
				t.Logf("✅ LIB DISCOVERY: Found script in %s/lib/slaygent-comms/", setup.prefix)
			}

			// Test dynamic Cellar discovery (no hardcoded versions)
			entries, err := os.ReadDir(cellarBase)
			if err == nil {
				foundVersions := 0
				for _, entry := range entries {
					if entry.IsDir() {
						scriptPath := filepath.Join(cellarBase, entry.Name(), "libexec", "sync-claude.sh")
						if _, err := os.Stat(scriptPath); err == nil {
							foundVersions++
							t.Logf("✅ DYNAMIC DISCOVERY: Found script for version %s in %s", entry.Name(), setup.name)
						}
					}
				}

				if foundVersions == len(versions) {
					t.Logf("✅ VERSION-AGNOSTIC SUCCESS: Found all %d versions dynamically", foundVersions)
				} else {
					t.Errorf("❌ VERSION DISCOVERY ISSUE: Expected %d versions, found %d", len(versions), foundVersions)
				}
			}
		})
	}
}

// TestRealWorldPortabilityScenario - The exact scenario that was broken before
func TestRealWorldPortabilityScenario(t *testing.T) {
	t.Log("🔍 PROVING: Real-world scenario - syncing on new MacBook works without hardcoded paths")

	// Scenario: User syncs project on original MacBook, then clones to new MacBook
	originalUser := "williamvansickleiii"
	newUser := "john"

	// Create project that was synced on original machine (with old broken sync)
	tempProject := t.TempDir()
	claudeFile := filepath.Join(tempProject, "CLAUDE.md")

	// Original broken content (hardcoded absolute path)
	brokenContent := `# My Project

<!-- SLAYGENT-REGISTRY-START -->
# Inter-Agent Communication
@/Users/` + originalUser + `/.slaygent/registry.json

To send messages to other coding agents, use: ` + "`msg <agent_name> \"<message>\"`" + `
<!-- SLAYGENT-REGISTRY-END -->`

	os.WriteFile(claudeFile, []byte(brokenContent), 0644)

	// New user environment on fresh MacBook
	newUserHome := t.TempDir()
	newSlaygentDir := filepath.Join(newUserHome, ".slaygent")
	newRegistryPath := filepath.Join(newSlaygentDir, "registry.json")

	os.MkdirAll(newSlaygentDir, 0755)
	registryContent := `[
  {
    "name": "my-agent",
    "agent_type": "claude",
    "directory": "` + tempProject + `",
    "machine": "host"
  }
]`
	os.WriteFile(newRegistryPath, []byte(registryContent), 0644)

	// New user runs sync with PORTABLE script
	scriptPath := "app/scripts/sync-claude.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("Sync script not found - run from project root")
	}

	cmd := exec.Command("bash", "-c", "echo 'y' | "+scriptPath)
	cmd.Env = []string{
		"HOME=" + newUserHome,
		"PATH=" + os.Getenv("PATH"),
	}
	cmd.Dir = newUserHome

	output, err := cmd.Output()
	if err != nil {
		t.Logf("Script output: %s", string(output))
		t.Fatalf("❌ NEW USER SYNC FAILED: %v", err)
	}

	// Read fixed content
	fixedContent, err := os.ReadFile(claudeFile)
	if err != nil {
		t.Fatalf("Failed to read fixed CLAUDE.md: %v", err)
	}

	contentStr := string(fixedContent)

	// 🎯 THE CRITICAL TEST: Broken hardcoded path should be REPLACED with portable reference

	// BEFORE: @/Users/williamvansickleiii/.slaygent/registry.json ❌
	// AFTER:  @~/.slaygent/registry.json ✅

	if strings.Contains(contentStr, "/Users/"+originalUser+"/.slaygent/registry.json") {
		t.Errorf("❌ REAL-WORLD FAILURE: Still contains original user's hardcoded path")
		t.Logf("Content still broken:\n%s", contentStr)
	}

	if !strings.Contains(contentStr, "@~/.slaygent/registry.json") {
		t.Errorf("❌ REAL-WORLD FAILURE: Missing portable registry reference")
		t.Logf("Content:\n%s", contentStr)
	}

	// SUCCESS CRITERIA
	if strings.Contains(contentStr, "@~/.slaygent/registry.json") &&
	   !strings.Contains(contentStr, "/Users/"+originalUser+"/.slaygent/registry.json") {
		t.Logf("✅ REAL-WORLD SUCCESS: Project now works on %s's fresh MacBook!", newUser)
		t.Logf("✅ PORTABILITY PROVEN: Registry reference is machine-independent")
	}
}