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
	// m=<media> <port> <transport> <payload types>
	// - media: video or audio
	// - port: RTP port
	// - transport: transport protocol RTP/AVP or UDP
	// - payload types: 97

	fields := strings.Fields(line)
	if len(fields) < 4 {
		return nil
	}

	port, err := strconv.Atoi(fields[1])
	if err != nil {
		return nil
	}

	format, err := strconv.Atoi(fields[3])
	if err != nil {
		return nil
	}

	return &SdpItem{
		Port:      port,
		Transport: fields[2],
		Format:    format,
	}
}

func (m *SdpItem) parse_a_rtpmap(line string) {
	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<channels>]

	fields := strings.Fields(line)
	if len(fields) < 2 {
		return
	}

	params := strings.Split(fields[1], "/")
	if len(params) < 2 {
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

func parse_a_control(control *url.URL, line string) (*url.URL, error) {
	line = strings.TrimSpace(line)

	// do nothing keep base URL
	if line == "*" {
		return control, nil
	}

	if strings.HasPrefix(line, "rtsp://") {
		return url.Parse(line)
	}

	path := control.Path
	if !strings.HasSuffix(path, "/") && !strings.HasPrefix(line, "/") {
		path += "/"
	}
	path += line

	return control.Parse(path)
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
				if parsed, err := parse_a_control(control, value); err == nil {
					if item != nil {
						item.URL = parsed
					} else {
						// redefine base URL if a=control before m=
						control = parsed
					}
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
