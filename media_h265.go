package rtsp

import (
	"encoding/base64"
	"strconv"
	"strings"
)

// RTP Payload Format for High Efficiency Video Coding (HEVC)
// https://datatracker.ietf.org/doc/html/rfc7798
type MediaH265 struct {
	ClockRate int
	LevelID   int
	VPS       []byte
	SPS       []byte
	PPS       []byte
}

func NewMediaH265(clockRate int) *MediaH265 {
	return &MediaH265{
		ClockRate: clockRate,
	}
}

func (m *MediaH265) ParseFMTP(line string) {
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
		case "level-id":
			if v, err := strconv.Atoi(value); err == nil {
				m.LevelID = v
			}

		case "sprop-vps":
			if v, err := base64.StdEncoding.DecodeString(value); err == nil {
				m.VPS = v
			}

		case "sprop-sps":
			if v, err := base64.StdEncoding.DecodeString(value); err == nil {
				m.SPS = v
			}

		case "sprop-pps":
			if v, err := base64.StdEncoding.DecodeString(value); err == nil {
				m.PPS = v
			}
		}
	}
}
