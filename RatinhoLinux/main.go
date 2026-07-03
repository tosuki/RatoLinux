package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"ratinholinux/app"
)

func main() {
	toggleFlag := flag.Bool("toggle", false, "Alterna a visibilidade da instancia em execucao")
	flag.Parse()

	if *toggleFlag {
		// Tenta enviar o sinal de toggle para a instância em execução
		if app.SendToggleSignal() {
			fmt.Println("Sinal de alternancia de visibilidade enviado para a instancia existente.")
			os.Exit(0)
		} else {
			fmt.Println("Nenhuma instancia em execucao encontrada. Iniciando nova instancia...")
		}
	} else {
		// Se não foi chamado com --toggle, mas já existe uma instância rodando,
		// envia o toggle para ela e fecha (comportamento de instância única).
		if app.SendToggleSignal() {
			fmt.Println("Instancia do Ratinho ja esta rodando. Alternando visibilidade.")
			os.Exit(0)
		}
	}

	// Carrega configurações persistidas
	settings := LoadSettings()

	// Cria a instância do Game
	game := app.NewGame(app.AppSettings{
		PetID:                settings.PetID,
		Size:                 settings.Size,
		SqueakEnabled:        settings.SqueakEnabled,
		MusicEnabled:         settings.MusicEnabled,
		AudioReactiveEnabled: settings.AudioReactiveEnabled,
		Opacity:              settings.Opacity,
		Topmost:              settings.Topmost,
	})

	// Inicializa o servidor IPC socket para escutar comandos --toggle
	ipcCloser, err := app.StartIPCServer(func() {
		game.ToggleVisibility()
	})
	if err != nil {
		log.Printf("Aviso: Nao foi possivel iniciar servidor IPC local: %v\n", err)
	} else {
		game.SetIPCCloser(ipcCloser)
		defer ipcCloser.Close()
	}

	// Configuração do Ebitengine
	ebiten.SetScreenTransparent(true)
	ebiten.SetWindowDecorated(false)
	ebiten.SetWindowFloating(settings.Topmost)
	ebiten.SetWindowSize(int(settings.Size)+10, int(settings.Size)+10)
	ebiten.SetWindowTitle("Ratinho Desktop")

	// Inicia o loop principal do jogo
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
