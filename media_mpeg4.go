package rtsp

import (
	"encoding/hex"
	"strconv"
	"strings"
)

// RTP Payload Format for Transport of MPEG-4 Elementary Streams
// https://datatracker.ietf.org/doc/html/rfc3640
type MediaMPEG4 struct {
	ClockRate      int
	Mode           string
	ProfileLevelID int
	Config         []byte
}

func NewMediaMPEG4(clockRate int) *MediaMPEG4 {
	return &MediaMPEG4{
		ClockRate: clockRate,
	}
}

func (m *MediaMPEG4) ParseFMTP(line string) {
	var (
		pair, key, value string
		ok               bool
	)

	for line != "" {
		pair, line, _ = strings.Cut(line, ";")
		pair = strings.TrimSpace(pair)

		key, value, ok = strings.Cut(pair, "=")
		if !ok {
			continue
		}

		switch key {
		case "mode":
			m.Mode = value

		case "profile-level-id":
			if v, err := strconv.Atoi(value); err == nil {
				m.ProfileLevelID = v
			}

		case "config":
			if v, err := hex.DecodeString(value); err == nil {
				m.Config = v
			}
		}
	}
}
