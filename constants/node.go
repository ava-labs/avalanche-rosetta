package constants

import "errors"

var (
	ErrInvalidMode          = errors.New("invalid rosetta mode")
	ErrInvalidIngestionMode = errors.New("invalid rosetta ingestion mode")
)

type NodeMode uint8

const (
	Unknown NodeMode = iota + 1
	Offline
	Online
)

func (m NodeMode) String() string {
	switch m {
	case Offline:
		return "offline"
	case Online:
		return "online"
	default:
		return "unknown"
	}
}

func GetNodeMode(s string) (NodeMode, error) {
	switch {
	case s == "offline":
		return Offline, nil
	case s == "online":
		return Online, nil
	default:
		return Unknown, ErrInvalidMode
	}
}

type NodeIngestion uint8

const (
	UnknownIngestion NodeIngestion = iota + 1
	StandardIngestion
	AnalyticsIngestion
)

func (m NodeIngestion) String() string {
	switch m {
	case StandardIngestion:
		return "standard"
	case AnalyticsIngestion:
		return "analytics"
	default:
		return "unknown"
	}
}

func GetNodeIngestion(s string) (NodeIngestion, error) {
	switch {
	case s == "standard":
		return StandardIngestion, nil
	case s == "analytics":
		return AnalyticsIngestion, nil
	default:
		return UnknownIngestion, ErrInvalidIngestionMode
	}
}
