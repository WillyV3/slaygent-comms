package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// This is a copy of the findSyncScript logic to test it independently
func findSyncScript(scriptName string) string {
	// Try relative path (development)
	relativePath := "../scripts/" + scriptName
	if _, err := os.Stat(relativePath); err == nil {
		return relativePath
	}

	// Try dynamic Homebrew prefix detection first
	if brewPrefix := getHomebrewPrefix(); brewPrefix != "" {
		brewPaths := []string{
			filepath.Join(brewPrefix, "lib", "slaygent-comms", scriptName),                         // lib location
			filepath.Join(brewPrefix, "Cellar", "slaygent-comms", "*", "libexec", scriptName),     // cellar location with wildcard
		}

		for _, brewPath := range brewPaths {
			// Handle wildcard in cellar path
			if strings.Contains(brewPath, "*") {
				cellarBase := filepath.Join(brewPrefix, "Cellar", "slaygent-comms")
				if entries, err := os.ReadDir(cellarBase); err == nil {
					for _, entry := range entries {
						if entry.IsDir() {
							dynamicPath := filepath.Join(cellarBase, entry.Name(), "libexec", scriptName)
							if _, err := os.Stat(dynamicPath); err == nil {
								return dynamicPath
							}
						}
					}
				}
			} else {
				if _, err := os.Stat(brewPath); err == nil {
					return brewPath
				}
			}
		}
	}

	// Fallback to hardcoded common locations
	possiblePaths := []string{
		"/opt/homebrew/lib/slaygent-comms/" + scriptName,                                           // macOS ARM Homebrew
		"/usr/local/lib/slaygent-comms/" + scriptName,                                              // macOS Intel Homebrew
		"/home/linuxbrew/.linuxbrew/lib/slaygent-comms/" + scriptName,                              // Linux Homebrew (lib)
		"/usr/lib/slaygent-comms/" + scriptName,                                                    // System install
	}

	// Dynamic version detection for Cellar paths (no hardcoded versions)
	cellarBases := []string{
		"/opt/homebrew/Cellar/slaygent-comms",                                                      // macOS ARM
		"/usr/local/Cellar/slaygent-comms",                                                         // macOS Intel
		"/home/linuxbrew/.linuxbrew/Cellar/slaygent-comms",                                         // Linux
	}

	for _, cellarBase := range cellarBases {
		if entries, err := os.ReadDir(cellarBase); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					dynamicPath := filepath.Join(cellarBase, entry.Name(), "libexec", scriptName)
					if _, err := os.Stat(dynamicPath); err == nil {
						return dynamicPath
					}
				}
			}
		}
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fallback to relative path (will fail but with clear error)
	return relativePath
}

func getHomebrewPrefix() string {
	cmd := exec.Command("brew", "--prefix")
	cmd.Env = os.Environ()
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func main() {
	fmt.Println("🔍 TESTING SCRIPT DISCOVERY LOGIC")
	fmt.Println("==================================")

	// Test Homebrew prefix detection
	brewPrefix := getHomebrewPrefix()
	fmt.Printf("Homebrew prefix: %s\n", brewPrefix)
	if brewPrefix == "" {
		fmt.Println("❌ WARNING: Could not detect Homebrew prefix")
	}

	// Test script discovery
	scriptPath := findSyncScript("sync-claude.sh")
	fmt.Printf("Script path found: %s\n", scriptPath)

	// Test if script actually exists
	if _, err := os.Stat(scriptPath); err == nil {
		fmt.Printf("✅ SUCCESS: Script exists at %s\n", scriptPath)

		// Test if script is executable
		if info, err := os.Stat(scriptPath); err == nil {
			mode := info.Mode()
			if mode&0111 != 0 {
				fmt.Println("✅ SUCCESS: Script is executable")
			} else {
				fmt.Println("❌ WARNING: Script is not executable")
			}
		}
	} else {
		fmt.Printf("❌ FAILURE: Script not found at %s\n", scriptPath)
		fmt.Printf("   Error: %v\n", err)
	}

	fmt.Println("\n📋 TROUBLESHOOTING INFO:")
	fmt.Println("If script discovery fails on another machine:")
	fmt.Println("1. Ensure Homebrew is installed: brew --version")
	fmt.Println("2. Ensure slaygent-comms is installed: brew list | grep slaygent")
	fmt.Println("3. Check installation: brew --prefix slaygent-comms")
	fmt.Println("4. Reinstall if needed: brew reinstall slaygent-comms")
}