package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const frameLimit = 256 * 1024

var extensionOrigin = regexp.MustCompile(`^chrome-extension://[a-p]{32}/$`)

type endpoint struct {
	Port  int    `json:"port"`
	Token string `json:"token"`
}

type envelope struct {
	Type         string          `json:"type"`
	ConnectionID string          `json:"connectionId,omitempty"`
	Message      json.RawMessage `json:"message,omitempty"`
	Protocol     int             `json:"protocol,omitempty"`
	Version      string          `json:"version,omitempty"`
}

func readFrame(reader io.Reader) ([]byte, error) {
	var header [4]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, err
	}
	size := binary.NativeEndian.Uint32(header[:])
	if size == 0 || size > frameLimit {
		return nil, errors.New("invalid frame size")
	}
	body := make([]byte, size)
	_, err := io.ReadFull(reader, body)
	return body, err
}

func writeFrame(writer io.Writer, body []byte) error {
	if len(body) == 0 || len(body) > frameLimit {
		return errors.New("invalid frame size")
	}
	data := make([]byte, 4+len(body))
	binary.NativeEndian.PutUint32(data[:4], uint32(len(body)))
	copy(data[4:], body)
	n, err := writer.Write(data)
	if err == nil && n != len(data) {
		return io.ErrShortWrite
	}
	return err
}

func endpointPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "AIbertAI Browser Host", "endpoint.json"), nil
}

func publishEndpoint(path string, address endpoint) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), "endpoint-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if err := errors.Join(json.NewEncoder(file).Encode(address), file.Close()); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func readEndpoint(path string) (endpoint, error) {
	var address endpoint
	file, err := os.Open(path)
	if err != nil {
		return address, err
	}
	defer file.Close()
	err = json.NewDecoder(io.LimitReader(file, 1024)).Decode(&address)
	if err != nil || address.Port < 1 || address.Port > 65535 || len(address.Token) != 64 {
		return endpoint{}, errors.New("invalid host endpoint")
	}
	return address, nil
}

type peer struct {
	socket net.Conn
	out    sync.Mutex
}

func (p *peer) send(body []byte) {
	p.out.Lock()
	defer p.out.Unlock()
	_ = p.socket.SetWriteDeadline(time.Now().Add(2 * time.Second))
	if writeFrame(p.socket, body) != nil {
		_ = p.socket.Close()
	}
}

func serve(path string, stdin io.Reader, stdout io.Writer) error {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return err
	}
	defer listener.Close()
	address := endpoint{Port: listener.Addr().(*net.TCPAddr).Port, Token: hex.EncodeToString(randBytes(32))}
	if err := publishEndpoint(path, address); err != nil {
		return err
	}
	defer func() {
		if current, err := readEndpoint(path); err == nil && current == address {
			_ = os.Remove(path)
		}
	}()
	var out sync.Mutex
	emit := func(event envelope) error {
		body, err := json.Marshal(event)
		if err != nil {
			return err
		}
		out.Lock()
		defer out.Unlock()
		return writeFrame(stdout, body)
	}
	if err := emit(envelope{Type: "ready", Protocol: 4, Version: "1.0.0"}); err != nil {
		return err
	}
	var peers sync.Map
	defer peers.Range(func(_, value any) bool {
		_ = value.(*peer).socket.Close()
		return true
	})
	slots := make(chan struct{}, 8)
	go func() {
		for {
			socket, err := listener.Accept()
			if err != nil {
				return
			}
			select {
			case slots <- struct{}{}:
			default:
				_ = socket.Close()
				continue
			}
			go func(socket net.Conn) {
				defer socket.Close()
				defer func() { <-slots }()
				_ = socket.SetReadDeadline(time.Now().Add(2 * time.Second))
				token, err := readFrame(socket)
				if err != nil || subtle.ConstantTimeCompare(token, []byte(address.Token)) != 1 {
					return
				}
				_ = socket.SetReadDeadline(time.Time{})
				id := hex.EncodeToString(randBytes(16))
				peers.Store(id, &peer{socket: socket})
				defer peers.Delete(id)
				if emit(envelope{Type: "connect", ConnectionID: id}) != nil {
					return
				}
				defer emit(envelope{Type: "disconnect", ConnectionID: id})
				for {
					message, err := readFrame(socket)
					if err != nil || !json.Valid(message) || emit(envelope{Type: "message", ConnectionID: id, Message: message}) != nil {
						return
					}
				}
			}(socket)
		}
	}()
	for {
		body, err := readFrame(stdin)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var command envelope
		if err := json.Unmarshal(body, &command); err != nil || command.Type != "send" || len(command.Message) == 0 {
			return errors.New("invalid desktop message")
		}
		if value, ok := peers.Load(command.ConnectionID); ok {
			value.(*peer).send(command.Message)
		}
	}
}

func randBytes(size int) []byte {
	data := make([]byte, size)
	rand.Read(data)
	return data
}

func relay(path string, stdin io.Reader, stdout io.Writer) error {
	address, err := readEndpoint(path)
	if err != nil {
		return err
	}
	socket, err := net.DialTimeout("tcp4", fmt.Sprintf("127.0.0.1:%d", address.Port), 2*time.Second)
	if err != nil {
		return err
	}
	defer socket.Close()
	if err := writeFrame(socket, []byte(address.Token)); err != nil {
		return err
	}
	go func() {
		for {
			body, err := readFrame(stdin)
			if err != nil || len(body) > frameLimit-256 || writeFrame(socket, body) != nil {
				_ = socket.Close()
				return
			}
		}
	}()
	for {
		body, err := readFrame(socket)
		if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := writeFrame(stdout, body); err != nil {
			return err
		}
	}
}

func main() {
	path, err := endpointPath()
	if err == nil {
		switch {
		case len(os.Args) == 2 && os.Args[1] == "--listen":
			err = serve(path, os.Stdin, os.Stdout)
		case len(os.Args) >= 2 && extensionOrigin.MatchString(os.Args[1]):
			err = relay(path, os.Stdin, os.Stdout)
		default:
			err = errors.New("unsupported browser host invocation")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "AIbertAI browser host:", err)
		os.Exit(1)
	}
}
