package audio

import (
	"bytes"
	"io"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

var (
	audioCtx      *audio.Context
	audioCtxOnce  sync.Once
	musicPlayer   *audio.Player
	squeakEnabled = true
	musicEnabled  = false
	mu            sync.Mutex
)

// InitAudio inicializa o contexto global de áudio do Ebitengine.
func InitAudio() {
	audioCtxOnce.Do(func() {
		// Ebiten context rodando a 44100Hz
		audioCtx = audio.NewContext(44100)
	})
}

// GetAudioContext retorna o contexto de áudio inicializado.
func GetAudioContext() *audio.Context {
	InitAudio()
	return audioCtx
}

// SetSqueakEnabled ativa ou desativa o efeito sonoro de clique.
func SetSqueakEnabled(enabled bool) {
	mu.Lock()
	squeakEnabled = enabled
	mu.Unlock()
}

// SetMusicEnabled ativa ou desativa a música de fundo.
func SetMusicEnabled(enabled bool) {
	mu.Lock()
	musicEnabled = enabled
	mu.Unlock()

	if enabled {
		StartMusic()
	} else {
		StopMusic()
	}
}

// PlayClickSound gera e toca o som associado ao pet se os efeitos estiverem ativos.
func PlayClickSound(char SoundCharacter) {
	mu.Lock()
	enabled := squeakEnabled
	mu.Unlock()

	if !enabled {
		return
	}

	ctx := GetAudioContext()
	wavBytes := GenerateWavBytes(char)
	stream, err := wav.DecodeWithContext(ctx, bytes.NewReader(wavBytes))
	if err != nil {
		return
	}

	player, err := ctx.NewPlayer(stream)
	if err != nil {
		return
	}
	player.SetVolume(0.8)
	player.Play()

	// Goroutine simples para fechar o player depois que terminar de tocar
	go func() {
		// Aguarda o término da reprodução antes de dar Close (ou deixa o GC cuidar)
		for player.IsPlaying() {
			// Pequeno sleep sem travar o loop
			// (Alternativamente o Ebiten fecha players órfãos no GC, mas é boa prática liberar)
		}
	}()
}

// StartMusic inicia ou retoma a melodia chiptune em loop infinito.
func StartMusic() {
	ctx := GetAudioContext()

	mu.Lock()
	defer mu.Unlock()

	if musicPlayer != nil {
		if !musicPlayer.IsPlaying() {
			musicPlayer.Play()
		}
		return
	}

	// Gera o WAV da melodia e decodifica
	wavBytes := GenerateMelodyWavBytes()
	stream, err := wav.DecodeWithContext(ctx, bytes.NewReader(wavBytes))
	if err != nil {
		return
	}

	// Cria o stream de loop infinito
	loopStream := audio.NewInfiniteLoop(stream, stream.Length())

	player, err := ctx.NewPlayer(loopStream)
	if err != nil {
		return
	}
	player.SetVolume(0.25) // Volume confortável padrão
	musicPlayer = player
	musicPlayer.Play()
}

// StopMusic pausa a melodia de fundo.
func StopMusic() {
	mu.Lock()
	defer mu.Unlock()

	if musicPlayer != nil && musicPlayer.IsPlaying() {
		musicPlayer.Pause()
	}
}

// SetMusicVolume ajusta o volume da música de fundo (0.0 a 1.0).
func SetMusicVolume(vol float64) {
	mu.Lock()
	defer mu.Unlock()

	if musicPlayer != nil {
		musicPlayer.SetVolume(vol)
	}
}
