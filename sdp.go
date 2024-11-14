package rtsp

import (
	"bufio"
	"bytes"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// SDP: Session Description Protocol
// https://datatracker.ietf.org/doc/html/rfc2327
type SdpItem struct {
	Media     Media
	Port      int
	Transport string
	Format    int
	URL       *url.URL
}

const SdpMimeType = "application/sdp"

func parse_m(line string) *SdpItem {
	// m=video 5006 RTP/AVP 97
	// - media: video
	// - port: 5006
	// - transport: RTP/AVP
	// - format: 97
	params := strings.Split(line, " ")

	if len(params) != 4 {
		return nil
	}

	port, err := strconv.Atoi(params[1])
	if err != nil {
		return nil
	}

	format, err := strconv.Atoi(params[3])
	if err != nil {
		return nil
	}

	return &SdpItem{
		Port:      port,
		Transport: params[2],
		Format:    format,
	}
}

func (m *SdpItem) parse_a_rtpmap(line string) {
	_, rtpmap, ok := strings.Cut(line, " ")
	if !ok {
		return
	}

	rtpmap = strings.TrimSpace(rtpmap)

	params := strings.Split(rtpmap, "/")
	if len(rtpmap) < 2 {
		return
	}

	clockRate, err := strconv.Atoi(params[1])
	if err != nil {
		return
	}

	m.Media = NewMedia(params[0], clockRate)
}

func (m *SdpItem) parse_a_fmtp(line string) {
	if m.Media == nil {
		return
	}

	if _, fmtp, ok := strings.Cut(line, " "); ok {
		m.Media.ParseFMTP(fmtp)
	}
}

// ParseSDP parses SDP from the bytes array.
// Returns list of SDP Items or error.
func ParseSDP(control *url.URL, data []byte) ([]*SdpItem, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty")
	}

	var (
		result []*SdpItem
		item   *SdpItem
	)

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		switch line[0] {
		case 'm':
			item = parse_m(line[2:])
			if item != nil {
				item.URL = control
				result = append(result, item)
			}

		case 'a':
			attr, value, ok := strings.Cut(line[2:], ":")
			if !ok {
				continue
			}

			switch attr {
			case "control":
				u := control
				if value != "*" {
					if parsed, err := url.Parse(value); err == nil {
						u = u.ResolveReference(parsed)
					}
				}

				if item == nil {
					control = u
				} else {
					item.URL = u
				}

			case "rtpmap":
				if item != nil {
					item.parse_a_rtpmap(value)
				}

			case "fmtp":
				if item != nil {
					item.parse_a_fmtp(value)
				}
			}
		}
	}

	return result, nil
}
