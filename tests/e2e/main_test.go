package e2e

import (
	"fmt"
	"os"
	"testing"
)

// TestMain handles setup and teardown for all E2E tests
func TestMain(m *testing.M) {
	// Check if we should skip E2E tests
	if os.Getenv("SKIP_E2E") == "true" {
		fmt.Println("Skipping E2E tests (SKIP_E2E=true)")
		return
	}

	fmt.Println("🚀 Starting E2E test setup...")

	// Wait for the Posts service to be ready
	fmt.Println("⏳ Waiting for Posts service to be ready...")
	WaitForService(&testing.T{}, "http://localhost:8081/health", 30)
	fmt.Println("✅ Posts service is ready")

	// Additional setup if needed
	setupTestData()

	// Run tests
	fmt.Println("🧪 Running E2E tests...")
	code := m.Run()

	// Cleanup
	fmt.Println("🧹 Cleaning up test data...")
	cleanupTestData()

	fmt.Println("✅ E2E tests completed")
	os.Exit(code)
}

func setupTestData() {
	// Any global test data setup can go here
	// For now, we'll create test data in individual tests
}

func cleanupTestData() {
	// Global cleanup if needed
	// Individual tests should clean up their own data
}

