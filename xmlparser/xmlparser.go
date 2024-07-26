package xmlparser

import (
	"commi/commit"
	"encoding/xml"
	"fmt"
	"strings"
)

type XMLCommit struct {
	XMLName xml.Name `xml:"commit"`
	Title   string   `xml:"title"`
	Changes struct {
		Items []string `xml:"change"`
	} `xml:"changes"`
	Summary string `xml:"summary"`
}

func ParseXMLCommit(xmlContent string) (*commit.Commit, error) {
	var xmlCommit XMLCommit
	if err := xml.Unmarshal([]byte(xmlContent), &xmlCommit); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %v", err)
	}

	message := strings.Join(xmlCommit.Changes.Items, "\n")
	summary := strings.TrimSpace(xmlCommit.Summary)

	if message != "" && summary != "" {
		message += "\n\n"
	}
	message += summary

	return &commit.Commit{
		Title:   xmlCommit.Title,
		Message: strings.TrimSpace(message),
	}, nil
}
