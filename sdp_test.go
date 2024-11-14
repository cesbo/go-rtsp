package rtsp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSdp_ParseSDP(t *testing.T) {
	assert := assert.New(t)

	data := []byte(strings.Join(
		[]string{
			`v=0`,
			`m=audio 5004 RTP/AVP 96`,
			`a=rtpmap:96 mpeg4-generic/8000/2`,
			`a=fmtp:96 streamtype=5; profile-level-id=15; mode=AAC-hbr; config=1588`,
			`a=control:trackID=0`,
			`m=video 5006 RTP/AVP 97`,
			`a=rtpmap:97 H264/90000`,
			`a=fmtp:97 profile-level-id=428014;sprop-parameter-sets=Z0KAFNoFB+Q=,aM4G4g==;`,
			`a=control:trackID=1`,
		},
		"\r\n",
	))
	u, _ := url.Parse("rtsp://test.com")
	s, err := ParseSDP(u, data)
	if !assert.NoError(err) {
		return
	}

	c1, _ := url.Parse("trackID=0")
	c2, _ := url.Parse("trackID=1")

	expectedSdp := []*SdpItem{
		{
			Port:      5004,
			Transport: "RTP/AVP",
			Format:    96,
			URL:       u.ResolveReference(c1),

			Media: &MediaMPEG4{
				ClockRate:      8000,
				Mode:           "AAC-hbr",
				ProfileLevelID: 15,
				Config:         []byte{0x15, 0x88},
			},
		},
		{
			Port:      5006,
			Transport: "RTP/AVP",
			Format:    97,
			URL:       u.ResolveReference(c2),

			Media: &MediaH264{
				ClockRate:      90000,
				ProfileLevelID: []byte{0x42, 0x80, 0x14},
				SPS: []byte{
					0x67, 0x42, 0x80, 0x14, 0xda, 0x05, 0x07, 0xe4,
				},
				PPS: []byte{
					0x68, 0xce, 0x06, 0xe2,
				},
			},
		},
	}

	assert.Equal(expectedSdp, s)
}

func TestSdp_parse_a_control(t *testing.T) {
	t.Run("globa control", func(t *testing.T) {
		assert := assert.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"a=control:rtsp://example.com/movie/",
				"m=video 8002 RTP/AVP 31",
				"a=control:trackID=1",
				"m=audio 8004 RTP/AVP 3",
				"a=control:trackID=2",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://test.com")
		s, err := ParseSDP(u, data)
		if !assert.NoError(err) {
			return
		}

		control, _ := url.Parse("rtsp://example.com/movie/")
		c1, _ := url.Parse("trackID=1")
		c2, _ := url.Parse("trackID=2")

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       control.ResolveReference(c1),
			},
			{
				Port:      8004,
				Transport: "RTP/AVP",
				Format:    3,
				URL:       control.ResolveReference(c2),
			},
		}

		assert.Equal(expectedSdp, s)
	})

	t.Run("asterisk control", func(t *testing.T) {
		assert := assert.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"a=control:rtsp://example.com/movie/",
				"m=video 8002 RTP/AVP 31",
				"a=control:*",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://example.com/movie/")
		s, err := ParseSDP(u, data)
		assert.Equal(nil, err)

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       u,
			},
		}

		assert.Equal(expectedSdp, s)
	})

	t.Run("default control", func(t *testing.T) {
		assert := assert.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"m=video 8002 RTP/AVP 31",
				"a=control:*",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://example.com/movie/")
		s, err := ParseSDP(u, data)
		assert.Equal(nil, err)

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       u,
			},
		}

		assert.Equal(expectedSdp, s)
	})
}

// space in rtpmap line
func TestSdp_rtpmap_bug_1(t *testing.T) {
	assert := assert.New(t)

	data := []byte(strings.Join(
		[]string{
			`v=0`,
			`m=video 5006 RTP/AVP 97`,
			`a=rtpmap:97 H264/90000 `,
			`a=fmtp:97 profile-level-id=428014;sprop-parameter-sets=Z0KAFNoFB+Q=,aM4G4g==;`,
			`a=control:trackID=1`,
		},
		"\r\n",
	))
	u, _ := url.Parse("rtsp://test.com")
	s, err := ParseSDP(u, data)
	if !assert.NoError(err) {
		return
	}

	assert.Len(s, 1)
	_, ok := s[0].Media.(*MediaH264)
	assert.True(ok)
}
