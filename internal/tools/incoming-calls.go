package tools

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/isaacphi/mcp-language-server/internal/lsp"
	"github.com/isaacphi/mcp-language-server/internal/protocol"
)

func FindIncomingCalls(ctx context.Context, client *lsp.Client, symbolName string) (string, error) {
	// Get context lines from environment variable
	contextLines := 5
	if envLines := os.Getenv("LSP_CONTEXT_LINES"); envLines != "" {
		if val, err := strconv.Atoi(envLines); err == nil && val >= 0 {
			contextLines = val
		}
	}

	// First get the symbol location like ReadDefinition does
	symbolResult, err := client.Symbol(ctx, protocol.WorkspaceSymbolParams{
		Query: symbolName,
	})
	if err != nil {
		return "", fmt.Errorf("failed to fetch symbol: %v", err)
	}

	results, err := symbolResult.Results()
	if err != nil {
		return "", fmt.Errorf("failed to parse results: %v", err)
	}

	var allIncomingCalls []string
	for _, symbol := range results {
		// Handle different matching strategies based on the search term
		if strings.Contains(symbolName, ".") {
			// For qualified names like "Type.Method", check for various matches
			parts := strings.Split(symbolName, ".")
			methodName := parts[len(parts)-1]

			// Try matching the unqualified method name for languages that don't use qualified names in symbols
			if symbol.GetName() != symbolName && symbol.GetName() != methodName {
				continue
			}
		} else if symbol.GetName() != symbolName {
			// For unqualified names, exact match only
			continue
		}

		// Get the location of the symbol
		loc := symbol.GetLocation()

		// Open the file
		err := client.OpenFile(ctx, loc.URI.Path())
		if err != nil {
			toolsLogger.Error("Error opening file: %v", err)
			continue
		}

		// Prepare call hierarchy
		prepareParams := protocol.CallHierarchyPrepareParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{
					URI: loc.URI,
				},
				Position: loc.Range.Start,
			},
		}

		items, err := client.PrepareCallHierarchy(ctx, prepareParams)
		if err != nil {
			return "", fmt.Errorf("failed to prepare call hierarchy: %v", err)
		}

		if len(items) == 0 {
			continue
		}

		// Get incoming calls for each item
		for _, item := range items {
			incomingCallsParams := protocol.CallHierarchyIncomingCallsParams{
				Item: item,
			}

			incomingCalls, err := client.IncomingCalls(ctx, incomingCallsParams)
			if err != nil {
				return "", fmt.Errorf("failed to get incoming calls: %v", err)
			}

			if len(incomingCalls) == 0 {
				continue
			}

			// Group calls by file
			callsByFile := make(map[protocol.DocumentUri][]protocol.CallHierarchyIncomingCall)
			for _, call := range incomingCalls {
				callsByFile[call.From.URI] = append(callsByFile[call.From.URI], call)
			}

			// Get sorted list of URIs
			uris := make([]string, 0, len(callsByFile))
			for uri := range callsByFile {
				uris = append(uris, string(uri))
			}
			sort.Strings(uris)

			// Process each file's calls in sorted order
			for _, uriStr := range uris {
				uri := protocol.DocumentUri(uriStr)
				fileCalls := callsByFile[uri]
				filePath := strings.TrimPrefix(uriStr, "file://")

				// Format file header
				fileInfo := fmt.Sprintf("---\n\n%s\nIncoming Calls in File: %d\n",
					filePath,
					len(fileCalls),
				)

				// Format locations with context
				fileContent, err := os.ReadFile(filePath)
				if err != nil {
					// Log error but continue with other files
					allIncomingCalls = append(allIncomingCalls, fileInfo+"\nError reading file: "+err.Error())
					continue
				}

				lines := strings.Split(string(fileContent), "\n")

				// Track call locations for header display
				var locStrings []string
				var locations []protocol.Location
				for _, call := range fileCalls {
					// Add the caller location
					loc := protocol.Location{
						URI:   call.From.URI,
						Range: call.From.SelectionRange,
					}
					locations = append(locations, loc)

					locStr := fmt.Sprintf("L%d:C%d (%s)",
						call.From.SelectionRange.Start.Line+1,
						call.From.SelectionRange.Start.Character+1,
						call.From.Name)
					locStrings = append(locStrings, locStr)
				}

				// Collect lines to display using the utility function
				linesToShow, err := GetLineRangesToDisplay(ctx, client, locations, len(lines), contextLines)
				if err != nil {
					// Log error but continue with other files
					continue
				}

				// Convert to line ranges using the utility function
				lineRanges := ConvertLinesToRanges(linesToShow, len(lines))

				// Format with locations in header
				formattedOutput := fileInfo
				if len(locStrings) > 0 {
					formattedOutput += "Callers: " + strings.Join(locStrings, ", ") + "\n"
				}

				// Format the content with ranges
				formattedOutput += "\n" + FormatLinesWithRanges(lines, lineRanges)
				allIncomingCalls = append(allIncomingCalls, formattedOutput)
			}
		}
	}

	if len(allIncomingCalls) == 0 {
		return fmt.Sprintf("No incoming calls found for symbol: %s", symbolName), nil
	}

	return strings.Join(allIncomingCalls, "\n"), nil
}
