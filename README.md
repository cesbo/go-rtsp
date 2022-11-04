# go-rtsp

[![Go Reference](https://pkg.go.dev/badge/github.com/cesbo/go-rtsp.svg)](https://pkg.go.dev/github.com/cesbo/go-rtsp)

RTSP Client

Features:

- Authentication
    - Basic
    - Digest
- Transport
    - UDP unicast
    - TCP
- Media
    - mpeg4-generic
    - h.264
    - h.265

## Installation

To install the library use the following command in the project directory:

```
go get github.com/cesbo/go-rtsp
```

## Quick Start

```go
type Handler struct{}
func (Handler) OnRTP(mediaID int, packet []byte) {}
func (Handler) OnRTCP(mediaID int, packet []byte) {}

handler := &Handler{}
ctx := context.Background()
url, _ := url.Parse("rtsp://user:pass@localhost:554/live")

rtspClient := &rtsp.Client{
    UseTCP:         false,
    URL:            url,
}
defer rtspClient.Close()

_ = rtspClient.Start(ctx)

for mediaID, sdpItem := range rtspClient.GetSDP() {
    if sdpItem.Media == nil {
        continue
    }

    if err := rtspClient.Setup(ctx, mediaID, sdpItem.URL); err != nil {
        return err
    }
}

_ = rtspClient.Play(ctx, handler)
```
