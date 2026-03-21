package provider

import (
	"bufio"
	"bytes"
	"strings"
)

// SplitLLMSTxt parses the xAI-style llms.txt format into individual pages.
// The format uses `===/<path>===` delimiters between sections.
// Each section becomes a Page with the path derived from the delimiter.
//
// Example input:
//
//	===/overview===
//	# Welcome
//	Some content here.
//
//	===/developers/tools/overview===
//	# Tools Overview
//	More content.
func SplitLLMSTxt(content []byte, sourceURL string) []Page {
	var pages []Page
	var currentPath string
	var currentContent bytes.Buffer

	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "===/") && strings.HasSuffix(line, "===") {
			// Flush the previous section.
			if currentPath != "" {
				pages = append(pages, Page{
					SourceURL: sourceURL,
					Path:      pathToArchivePath(currentPath),
					Content:   bytes.TrimSpace(currentContent.Bytes()),
				})
			}

			// Start a new section.
			currentPath = strings.TrimPrefix(line, "===")
			currentPath = strings.TrimSuffix(currentPath, "===")
			currentContent.Reset()
			continue
		}

		if currentPath != "" {
			currentContent.WriteString(line)
			currentContent.WriteByte('\n')
		}
	}

	// Flush the last section.
	if currentPath != "" {
		pages = append(pages, Page{
			SourceURL: sourceURL,
			Path:      pathToArchivePath(currentPath),
			Content:   bytes.TrimSpace(currentContent.Bytes()),
		})
	}

	return pages
}
