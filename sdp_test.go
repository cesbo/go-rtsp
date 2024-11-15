package rtsp

import (
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSdp_ParseSDP(t *testing.T) {
	require := require.New(t)

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
	u, _ := url.Parse("rtsp://test.local")
	s, err := ParseSDP(u, data)
	require.NoError(err)

	c1, _ := url.Parse("rtsp://test.local/trackID=0")
	c2, _ := url.Parse("rtsp://test.local/trackID=1")

	expectedSdp := []*SdpItem{
		{
			Port:      5004,
			Transport: "RTP/AVP",
			Format:    96,
			URL:       c1,

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
			URL:       c2,

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

	require.Equal(expectedSdp, s)
}

func TestSdp_parse_a_control(t *testing.T) {
	t.Run("redefine base url", func(t *testing.T) {
		require := require.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"a=control:rtsp://test.local/setup/",
				"m=video 8002 RTP/AVP 31",
				"a=control:trackID=1",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://test.local/example/")
		s, err := ParseSDP(u, data)
		require.NoError(err)

		c1, _ := url.Parse("rtsp://test.local/setup/trackID=1")

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       c1,
			},
		}

		require.Equal(expectedSdp, s)
	})

	t.Run("asterisk control", func(t *testing.T) {
		require := require.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"a=control:rtsp://test.local/movie/",
				"m=video 8002 RTP/AVP 31",
				"a=control:*",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://test.local/movie/")
		s, err := ParseSDP(u, data)
		require.NoError(err)

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       u,
			},
		}

		require.Equal(expectedSdp, s)
	})

	t.Run("absolute control", func(t *testing.T) {
		require := require.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"m=video 8002 RTP/AVP 31",
				"a=control:rtsp://test.local/trackID=1",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://test.local/example")
		s, err := ParseSDP(u, data)
		require.NoError(err)

		c1, _ := url.Parse("rtsp://test.local/trackID=1")

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       c1,
			},
		}

		require.Equal(expectedSdp, s)
	})

	t.Run("relative control", func(t *testing.T) {
		require := require.New(t)

		data := []byte(strings.Join(
			[]string{
				"v=0",
				"m=video 8002 RTP/AVP 31",
				"a=control:trackID=1",
			},
			"\r\n",
		))
		u, _ := url.Parse("rtsp://test.local/user=&password=&channel=1&stream=0.sdp")
		s, err := ParseSDP(u, data)
		require.NoError(err)

		c1, _ := url.Parse("rtsp://test.local/user=&password=&channel=1&stream=0.sdp/trackID=1")

		expectedSdp := []*SdpItem{
			{
				Port:      8002,
				Transport: "RTP/AVP",
				Format:    31,
				URL:       c1,
			},
		}

		require.Equal(expectedSdp, s)
	})
}

func TestSdp_parse_a_rtpmap(t *testing.T) {
	t.Run("trailing space bug", func(t *testing.T) {
		require := require.New(t)

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
		u, _ := url.Parse("rtsp://test.local")
		s, err := ParseSDP(u, data)
		require.NoError(err)

		require.Len(s, 1)
		_, ok := s[0].Media.(*MediaH264)
		require.True(ok)
	})
}
