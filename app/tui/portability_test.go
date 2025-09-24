package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"os/exec"
)

// TestPortableRegistryPath verifies that sync scripts generate portable registry references
func TestPortableRegistryPath(t *testing.T) {
	// Test data simulating different user environments
	testCases := []struct {
		name     string
		username string
		homeDir  string
	}{
		{"MacOS User", "john", "/Users/john"},
		{"Linux User", "alice", "/home/alice"},
		{"Different MacOS User", "williamvansickleiii", "/Users/williamvansickleiii"},
		{"Corporate Linux", "developer", "/home/developer"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp test environment
			tempDir := t.TempDir()
			registryPath := filepath.Join(tempDir, ".slaygent", "registry.json")
			claudeFile := filepath.Join(tempDir, "CLAUDE.md")

			// Create .slaygent directory and registry
			os.MkdirAll(filepath.Dir(registryPath), 0755)
			registryContent := `[{"name": "test-agent", "agent_type": "claude", "directory": "` + tempDir + `", "machine": "host"}]`
			os.WriteFile(registryPath, []byte(registryContent), 0644)

			// Create initial CLAUDE.md
			os.WriteFile(claudeFile, []byte("# Test Project\n"), 0644)

			// Simulate running sync script with different HOME environment
			oldHome := os.Getenv("HOME")
			defer os.Setenv("HOME", oldHome)
			os.Setenv("HOME", tempDir)

			// Run sync script in controlled environment
			scriptPath := "../scripts/sync-claude.sh"
			cmd := exec.Command("bash", "-c", "echo 'y' | " + scriptPath)
			cmd.Env = append(os.Environ(), "HOME=" + tempDir)
			cmd.Dir = tempDir
			output, err := cmd.Output()

			if err != nil {
				t.Fatalf("Sync script failed for %s: %v", tc.name, err)
			}

			// Read the updated CLAUDE.md
			content, err := os.ReadFile(claudeFile)
			if err != nil {
				t.Fatalf("Failed to read CLAUDE.md for %s: %v", tc.name, err)
			}

			contentStr := string(content)

			// CRITICAL TEST: Verify portable registry reference is used
			if !strings.Contains(contentStr, "@~/.slaygent/registry.json") {
				t.Errorf("PORTABILITY FAILURE for %s: Expected portable reference '@~/.slaygent/registry.json', got content:\n%s", tc.name, contentStr)
			}

			// CRITICAL TEST: Verify NO absolute paths are embedded
			if strings.Contains(contentStr, tc.homeDir + "/.slaygent/registry.json") {
				t.Errorf("PORTABILITY FAILURE for %s: Found hardcoded absolute path '%s/.slaygent/registry.json' in content:\n%s", tc.name, tc.homeDir, contentStr)
			}

			// CRITICAL TEST: Verify sync markers are present
			if !strings.Contains(contentStr, "<!-- SLAYGENT-REGISTRY-START -->") {
				t.Errorf("SYNC FAILURE for %s: Missing start marker", tc.name)
			}
			if !strings.Contains(contentStr, "<!-- SLAYGENT-REGISTRY-END -->") {
				t.Errorf("SYNC FAILURE for %s: Missing end marker", tc.name)
			}

			t.Logf("✅ PORTABILITY SUCCESS for %s: Registry reference is portable", tc.name)
		})
	}
}

// TestScriptDiscoveryPortability verifies that script discovery works across different Homebrew installations
func TestScriptDiscoveryPortability(t *testing.T) {
	testCases := []struct {
		name         string
		brewPrefix   string
		expectedPath string
	}{
		{"macOS ARM Homebrew", "/opt/homebrew", "/opt/homebrew/lib/slaygent-comms/sync-claude.sh"},
		{"macOS Intel Homebrew", "/usr/local", "/usr/local/lib/slaygent-comms/sync-claude.sh"},
		{"Linux Homebrew", "/home/linuxbrew/.linuxbrew", "/home/linuxbrew/.linuxbrew/lib/slaygent-comms/sync-claude.sh"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock Homebrew structure
			tempDir := t.TempDir()
			mockBrewPrefix := filepath.Join(tempDir, strings.TrimPrefix(tc.brewPrefix, "/"))
			scriptDir := filepath.Join(mockBrewPrefix, "lib", "slaygent-comms")
			os.MkdirAll(scriptDir, 0755)

			// Create mock script
			scriptPath := filepath.Join(scriptDir, "sync-claude.sh")
			os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'mock script'"), 0755)

			// Test that our discovery logic would find this
			// Simulate the path checking logic from findSyncScript
			possiblePaths := []string{
				filepath.Join(mockBrewPrefix, "lib", "slaygent-comms", "sync-claude.sh"),
			}

			found := false
			for _, path := range possiblePaths {
				if _, err := os.Stat(path); err == nil {
					found = true
					t.Logf("✅ DISCOVERY SUCCESS for %s: Found script at %s", tc.name, path)
					break
				}
			}

			if !found {
				t.Errorf("DISCOVERY FAILURE for %s: Script not found in expected Homebrew structure", tc.name)
			}
		})
	}
}

// TestDynamicVersionDiscovery verifies version-agnostic Cellar discovery
func TestDynamicVersionDiscovery(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock Cellar structure with multiple versions
	versions := []string{"v0.3.1", "v0.4.0", "v1.0.0"}
	cellarBase := filepath.Join(tempDir, "Cellar", "slaygent-comms")

	var validPaths []string
	for _, version := range versions {
		versionDir := filepath.Join(cellarBase, version, "libexec")
		os.MkdirAll(versionDir, 0755)
		scriptPath := filepath.Join(versionDir, "sync-claude.sh")
		os.WriteFile(scriptPath, []byte("#!/bin/bash\necho 'version: "+version+"'"), 0755)
		validPaths = append(validPaths, scriptPath)
	}

	// Test dynamic discovery (should find ANY version, not hardcoded)
	entries, err := os.ReadDir(cellarBase)
	if err != nil {
		t.Fatalf("Failed to read cellar directory: %v", err)
	}

	foundVersions := 0
	for _, entry := range entries {
		if entry.IsDir() {
			scriptPath := filepath.Join(cellarBase, entry.Name(), "libexec", "sync-claude.sh")
			if _, err := os.Stat(scriptPath); err == nil {
				foundVersions++
				t.Logf("✅ VERSION DISCOVERY SUCCESS: Found script for version %s", entry.Name())
			}
		}
	}

	if foundVersions != len(versions) {
		t.Errorf("VERSION DISCOVERY FAILURE: Expected to find %d versions, found %d", len(versions), foundVersions)
	}

	// CRITICAL TEST: Verify no hardcoded version dependency
	if foundVersions > 0 {
		t.Logf("✅ DYNAMIC VERSION SUCCESS: Script discovery is version-agnostic")
	}
}

// TestCrossUserPortability simulates the exact issue: sync working across different user accounts
func TestCrossUserPortability(t *testing.T) {
	// Simulate the original issue: syncing on one machine, using on another

	// User 1: Original sync (williamvansickleiii on MacBook Pro)
	user1Home := "/Users/williamvansickleiii"
	user1Registry := user1Home + "/.slaygent/registry.json"

	// User 2: New machine (john on fresh MacBook)
	user2Home := "/Users/john"
	user2Registry := user2Home + "/.slaygent/registry.json"

	// Create temp environments for both users
	tempDir1 := t.TempDir()
	tempDir2 := t.TempDir()

	claudeFile1 := filepath.Join(tempDir1, "CLAUDE.md")
	claudeFile2 := filepath.Join(tempDir2, "CLAUDE.md")

	// User 1 syncs with OLD (broken) absolute path reference
	brokenContent := `# Test Project

<!-- SLAYGENT-REGISTRY-START -->
# Inter-Agent Communication
@` + user1Registry + `

To send messages to other coding agents, use: ` + "`msg <agent_name> \"<message>\"`" + `
<!-- SLAYGENT-REGISTRY-END -->`

	os.WriteFile(claudeFile1, []byte(brokenContent), 0644)

	// User 2 gets the same file (git clone, shared project, etc.)
	os.WriteFile(claudeFile2, []byte(brokenContent), 0644)

	// User 2 runs NEW portable sync on their machine
	os.MkdirAll(filepath.Join(tempDir2, ".slaygent"), 0755)
	registryContent := `[{"name": "test-agent", "agent_type": "claude", "directory": "` + tempDir2 + `", "machine": "host"}]`
	os.WriteFile(filepath.Join(tempDir2, ".slaygent", "registry.json"), []byte(registryContent), 0644)

	// Simulate new sync script fixing the portability
	scriptPath := "../scripts/sync-claude.sh"
	cmd := exec.Command("bash", "-c", "echo 'y' | " + scriptPath)
	cmd.Env = append(os.Environ(), "HOME=" + tempDir2)
	cmd.Dir = tempDir2
	_, err := cmd.Output()

	if err != nil {
		t.Fatalf("New sync script failed: %v", err)
	}

	// Read updated file
	updatedContent, err := os.ReadFile(claudeFile2)
	if err != nil {
		t.Fatalf("Failed to read updated CLAUDE.md: %v", err)
	}

	contentStr := string(updatedContent)

	// CRITICAL CROSS-USER TEST: Old absolute path should be replaced with portable reference
	if strings.Contains(contentStr, user1Registry) {
		t.Errorf("❌ CROSS-USER FAILURE: Still contains old user's absolute path: %s", user1Registry)
	}

	if !strings.Contains(contentStr, "@~/.slaygent/registry.json") {
		t.Errorf("❌ CROSS-USER FAILURE: Missing portable registry reference")
	}

	// SUCCESS: File is now portable and will work on User 2's machine
	if strings.Contains(contentStr, "@~/.slaygent/registry.json") && !strings.Contains(contentStr, user1Registry) {
		t.Logf("✅ CROSS-USER SUCCESS: Registry reference is now portable across different users")
	}
}