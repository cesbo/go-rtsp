package rtsp

type MediaHandler interface {
	OnRTP(mediaID int, packet []byte)
	OnRTCP(mediaID int, packet []byte)
}

type Transport interface {
	// Setup configures the transport.
	// Returns the transport protocol header for RTSP SETUP request
	Setup(mediaID int) (string, error)
	// Play starts the transport
	Play(handler MediaHandler)
	// Close closes the transport
	Close()
	// Err returns the transport error
	Err() <-chan error
}
