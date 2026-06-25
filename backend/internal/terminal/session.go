package terminal

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/coder/websocket"
	"golang.org/x/crypto/ssh"
)

// runSession requests an interactive PTY on the SSH client and bridges it to the
// WebSocket: browser JSON frames drive stdin/resize, PTY output is streamed back
// as binary frames. It returns when either side closes.
func runSession(parentCtx context.Context, conn *websocket.Conn, client *ssh.Client) error {
	ctx, cancel := context.WithCancel(parentCtx)
	defer cancel()

	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", defaultRows, defaultCols, modes); err != nil {
		return err
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		return err
	}

	output := &wsWriter{ctx: ctx, conn: conn}
	session.Stdout = output
	session.Stderr = output

	if err := session.Shell(); err != nil {
		return err
	}

	// The remote shell exited: stop the bridge.
	go func() {
		_ = session.Wait()
		cancel()
	}()

	// The browser drives stdin and resize until it disconnects.
	go func() {
		readClientMessages(ctx, conn, stdin, session)
		cancel()
	}()

	<-ctx.Done()
	return nil
}

func readClientMessages(ctx context.Context, conn *websocket.Conn, stdin io.Writer, session *ssh.Session) {
	for {
		messageType, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if messageType != websocket.MessageText {
			continue
		}

		var message clientMessage
		if err := json.Unmarshal(data, &message); err != nil {
			continue
		}

		switch message.Type {
		case msgStdin:
			if _, err := stdin.Write([]byte(message.Data)); err != nil {
				return
			}
		case msgResize:
			if message.Cols > 0 && message.Rows > 0 {
				_ = session.WindowChange(message.Rows, message.Cols)
			}
		}
	}
}

// writeStatus sends a best-effort JSON status frame to the browser.
func writeStatus(ctx context.Context, conn *websocket.Conn, message statusMessage) {
	payload, err := json.Marshal(message)
	if err != nil {
		return
	}

	_ = conn.Write(ctx, websocket.MessageText, payload)
}

// wsWriter forwards PTY output to the WebSocket as binary frames. The SSH client
// writes stdout and stderr from separate goroutines, so writes are serialized.
type wsWriter struct {
	ctx  context.Context
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.conn.Write(w.ctx, websocket.MessageBinary, p); err != nil {
		return 0, err
	}

	return len(p), nil
}
