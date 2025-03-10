package core

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type xmlCommit struct {
	XMLName     xml.Name `xml:"commit"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
}

func parseCommitMessage(xmlContent string) (*CommitMessage, error) {
	// Clean up the XML content
	xmlContent = strings.TrimSpace(xmlContent)
	if !strings.HasPrefix(xmlContent, "<commit>") {
		// Try to find the commit tag
		start := strings.Index(xmlContent, "<commit>")
		if start == -1 {
			return nil, fmt.Errorf("invalid XML format: missing <commit> tag")
		}
		xmlContent = xmlContent[start:]
	}

	var commit xmlCommit
	if err := xml.Unmarshal([]byte(xmlContent), &commit); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	return &CommitMessage{
		Title:   strings.TrimSpace(commit.Title),
		Message: strings.TrimSpace(commit.Description),
	}, nil
}
