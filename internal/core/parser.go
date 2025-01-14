package core

import (
	"encoding/xml"
	"strings"
)

type xmlCommit struct {
	XMLName xml.Name `xml:"commit"`
	Title   string   `xml:"title"`
	Changes struct {
		Items []string `xml:"change"`
	} `xml:"changes"`
	Summary string `xml:"summary"`
}

func (c *Core) parseCommitMessage(xmlContent string) (*CommitMessage, error) {
	var xmlCommit xmlCommit
	if err := xml.Unmarshal([]byte(xmlContent), &xmlCommit); err != nil {
		return nil, &ErrParsingCommit{
			Msg: "invalid XML format",
			Err: err,
		}
	}

	message := strings.Join(xmlCommit.Changes.Items, "\n")
	summary := strings.TrimSpace(xmlCommit.Summary)

	if message != "" && summary != "" {
		message += "\n\n"
	}
	message += summary

	return &CommitMessage{
		Title:   xmlCommit.Title,
		Message: strings.TrimSpace(message),
	}, nil
}
