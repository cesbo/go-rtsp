package rtsp

import (
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"strings"
)

// RTP Payload Format for H.264 Video
// https://datatracker.ietf.org/doc/html/rfc6184
type MediaH264 struct {
	ClockRate         int
	PacketizationMode int
	ProfileLevelID    []byte
	SPS               []byte
	PPS               []byte
}

func NewMediaH264(clockRate int) *MediaH264 {
	return &MediaH264{
		ClockRate: clockRate,
	}
}

func (m *MediaH264) ParseFMTP(line string) {
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
		case "packetization-mode":
			if v, err := strconv.Atoi(value); err == nil {
				m.PacketizationMode = v
			}

		case "profile-level-id":
			if v, err := hex.DecodeString(value); err == nil && len(v) == 3 {
				m.ProfileLevelID = v
			}

		case "sprop-parameter-sets":
			sps, pps, ok := strings.Cut(value, ",")
			if !ok {
				continue
			}

			if v, err := base64.StdEncoding.DecodeString(sps); err == nil {
				m.SPS = v
			}

			if v, err := base64.StdEncoding.DecodeString(pps); err == nil {
				m.PPS = v
			}
		}
	}
}
