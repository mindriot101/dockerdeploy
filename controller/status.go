package controller

type BuildStatus string

const (
	Successful BuildStatus = "success"
	Failure                = "failed"
	Skipped                = "skipped"
)
