package terminal

// clientMessage is a control/data frame sent by the browser over the WebSocket.
// It is always JSON. Terminal output flows the other way as binary frames.
type clientMessage struct {
	Type string `json:"type"`
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

const (
	msgStdin  = "stdin"
	msgResize = "resize"
)

// statusMessage is an out-of-band JSON frame sent to the browser to report
// session lifecycle events. Terminal bytes are sent as binary frames instead.
type statusMessage struct {
	Type    string `json:"type"`
	State   string `json:"state,omitempty"`
	Message string `json:"message,omitempty"`
}

// InfoResponse is the non-sensitive description of the configured target,
// used by the frontend to render target details and gate the feature.
type InfoResponse struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    string `json:"port"`
	User    string `json:"user"`
}

const (
	defaultCols = 80
	defaultRows = 24
	// maxMessageBytes caps a single inbound WS frame (keystrokes are tiny).
	maxMessageBytes = 1 << 16
)
