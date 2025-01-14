package core

import "fmt"

type ErrParsingCommit struct {
	Msg string
	Err error
}

func (e *ErrParsingCommit) Error() string {
	return fmt.Sprintf("failed to parse commit: %s: %v", e.Msg, e.Err)
}

func (e *ErrParsingCommit) Unwrap() error {
	return e.Err
}

type ErrGeneratingCommit struct {
	Msg string
	Err error
}

func (e *ErrGeneratingCommit) Error() string {
	return fmt.Sprintf("failed to generate commit: %s: %v", e.Msg, e.Err)
}

func (e *ErrGeneratingCommit) Unwrap() error {
	return e.Err
}
