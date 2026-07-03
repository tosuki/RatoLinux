package app

import (
	"io"
	"net"
	"os"
	"path/filepath"
)

const SocketName = "ratinhodesktop.sock"

// GetSocketPath retorna o caminho do socket na pasta temporária.
func GetSocketPath() string {
	return filepath.Join(os.TempDir(), SocketName)
}

// SendToggleSignal tenta se conectar à instância existente e enviar o sinal de alternar visibilidade.
// Retorna true se conseguiu enviar (significando que já havia uma instância rodando).
func SendToggleSignal() bool {
	socketPath := GetSocketPath()
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return false
	}
	defer conn.Close()

	_, _ = conn.Write([]byte("toggle"))
	return true
}

// StartIPCServer inicia um servidor de socket Unix local para escutar comandos de novas instâncias.
func StartIPCServer(onToggle func()) (io.Closer, error) {
	socketPath := GetSocketPath()

	// Remove socket antigo se houver resquício de crash anterior
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	go func() {
		defer listener.Close()
		defer os.Remove(socketPath)

		buf := make([]byte, 128)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			n, err := conn.Read(buf)
			if err == nil && string(buf[:n]) == "toggle" {
				onToggle()
			}
			conn.Close()
		}
	}()

	return listener, nil
}
