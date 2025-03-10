package xmlparser

import (
	"encoding/xml"
	"fmt"
	"strings"
)

type Commit struct {
	Title   string
	Message string
}

type XMLCommit struct {
	XMLName     xml.Name `xml:"commit"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	Changes     struct {
		Items []string `xml:"change"`
	} `xml:"changes"`
	Summary string `xml:"summary"`
}

func ParseXMLCommit(xmlContent string) (*Commit, error) {
	var xmlCommit XMLCommit
	if err := xml.Unmarshal([]byte(xmlContent), &xmlCommit); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	var message string

	// Check if we have a description tag
	if xmlCommit.Description != "" {
		// Use the description directly as the message
		message = strings.TrimSpace(xmlCommit.Description)
	} else {
		// Use the old format with changes and summary
		message = strings.Join(xmlCommit.Changes.Items, "\n")
		summary := strings.TrimSpace(xmlCommit.Summary)

		if message != "" && summary != "" {
			message += "\n\n"
		}
		message += summary
	}

	return &Commit{
		Title:   xmlCommit.Title,
		Message: strings.TrimSpace(message),
	}, nil
}
