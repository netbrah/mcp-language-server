package incoming_calls_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/isaacphi/mcp-language-server/integrationtests/tests/common"
	"github.com/isaacphi/mcp-language-server/integrationtests/tests/go/internal"
	"github.com/isaacphi/mcp-language-server/internal/tools"
)

// TestFindIncomingCalls tests the FindIncomingCalls tool with Go symbols
// that have callers in different files
func TestFindIncomingCalls(t *testing.T) {
	suite := internal.GetTestSuite(t)

	ctx, cancel := context.WithTimeout(suite.Context, 10*time.Second)
	defer cancel()

	tests := []struct {
		name          string
		symbolName    string
		expectedText  string
		expectedFiles int // Number of files where callers should be found
		snapshotName  string
	}{
		{
			name:          "Function called from multiple files",
			symbolName:    "HelperFunction",
			expectedText:  "ConsumerFunction",
			expectedFiles: 2, // consumer.go and another_consumer.go
			snapshotName:  "helper-function",
		},
		{
			name:          "Function called from same file",
			symbolName:    "FooBar",
			expectedText:  "main",
			expectedFiles: 1, // main.go
			snapshotName:  "foobar-function",
		},
		{
			name:          "Method with callers",
			symbolName:    "SharedStruct.Method",
			expectedText:  "ConsumerFunction",
			expectedFiles: 1, // consumer.go or another_consumer.go
			snapshotName:  "struct-method",
		},
		{
			name:          "No callers found",
			symbolName:    "SharedConstant",
			expectedText:  "No incoming calls found",
			expectedFiles: 0,
			snapshotName:  "no-callers",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the FindIncomingCalls tool
			result, err := tools.FindIncomingCalls(ctx, suite.Client, tc.symbolName)
			if err != nil {
				t.Fatalf("Failed to find incoming calls: %v", err)
			}

			// Check that the result contains relevant information
			if !strings.Contains(result, tc.expectedText) {
				t.Errorf("Incoming calls do not contain expected text: %s", tc.expectedText)
			}

			// Count how many different files are mentioned in the result
			fileCount := countFilesInResult(result)
			if tc.expectedFiles > 0 && fileCount < tc.expectedFiles {
				t.Errorf("Expected incoming calls in at least %d files, but found in %d files",
					tc.expectedFiles, fileCount)
			}

			// Use snapshot testing to verify exact output
			common.SnapshotTest(t, "go", "incoming_calls", tc.snapshotName, result)
		})
	}
}

// countFilesInResult counts the number of unique files mentioned in the result
func countFilesInResult(result string) int {
	fileMap := make(map[string]bool)

	// Any line containing "workspace" and ".go" is a file path
	for line := range strings.SplitSeq(result, "\n") {
		if strings.Contains(line, "workspace") && strings.Contains(line, ".go") {
			if !strings.Contains(line, "Incoming Calls in File") {
				fileMap[line] = true
			}
		}
	}

	return len(fileMap)
}
