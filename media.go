package rtsp

import "strings"

type Media interface {
	ParseFMTP(line string)
}

func NewMedia(mediaType string, clockRate int) Media {
	switch strings.ToLower(mediaType) {
	case "mpeg4-generic":
		return NewMediaMPEG4(clockRate)
	case "h264":
		return NewMediaH264(clockRate)
	case "h265":
		return NewMediaH265(clockRate)
	default:
		return nil
	}
}
