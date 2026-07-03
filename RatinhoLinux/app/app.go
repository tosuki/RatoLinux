package app

import (
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"ratinholinux/assets"
	"ratinholinux/audio"
)

// PetConfig define a estrutura no JSON pets.json
type PetConfig struct {
	ID          string  `json:"id"`
	DisplayName string  `json:"displayName"`
	BaseBPM     float64 `json:"baseBpm"`
	Sound       string  `json:"sound"`
}

type PetConfigWrapper struct {
	Pets []PetConfig `json:"pets"`
}

// PetDefinition contém a definição em memória de um pet e seu GIF decodificado.
type PetDefinition struct {
	ID          string
	DisplayName string
	BaseBPM     float64
	Sound       audio.SoundCharacter
	GifPlayer   *GifPlayer
}

// GifPlayer gerencia os frames e delays de um GIF.
type GifPlayer struct {
	Frames []*ebiten.Image
	Delays []int // Delay em centisegundos (10ms)
	Index  int
	Timer  float64
}

// AppSettings mapeado de config.go
type AppSettings struct {
	PetID                string  `json:"petId"`
	Size                 float64 `json:"size"`
	SqueakEnabled        bool    `json:"squeakEnabled"`
	MusicEnabled         bool    `json:"musicEnabled"`
	AudioReactiveEnabled bool    `json:"audioReactiveEnabled"`
	Opacity              float64 `json:"opacity"`
	Topmost              bool    `json:"topmost"`
}

// Game representa a aplicação principal do Ebiten.
type Game struct {
	settings       AppSettings
	settingsMutex  sync.Mutex
	catalog        []*PetDefinition
	currentPet     *PetDefinition
	
	// Estado físico do DVD Mode
	isDvdMode      bool
	vx, vy         float64
	flipSign       float64
	flashTimer     int // Efeito de colisão (opacity decay)

	// Arraste por mouse
	isDragging     bool
	dragStartX     int
	dragStartY     int

	// Double click detection
	lastClickTime  time.Time

	// Menu de contexto
	menu           *Menu
	menuOpen       bool

	// Reatividade de áudio
	reactiveService *audio.AudioReactiveService
	pulseScale      float64
	pulseDecayTimer int

	// IPC Socket Listener para --toggle
	ipcCloser      io.Closer

	// Controle de visibilidade
	visible        bool
}

func NewGame(initialSettings AppSettings) *Game {
	g := &Game{
		settings:   initialSettings,
		flipSign:   1.0,
		pulseScale: 1.0,
		visible:    true,
	}

	g.loadCatalog()
	g.selectPet(initialSettings.PetID, false)
	g.setupMenu()

	// Inicializa áudio reativo
	g.reactiveService = audio.NewAudioReactiveService(g.onBeatDetected)
	if initialSettings.AudioReactiveEnabled {
		g.reactiveService.Start()
	}

	// Aplica estados iniciais de som e música
	audio.SetSqueakEnabled(initialSettings.SqueakEnabled)
	audio.SetMusicEnabled(initialSettings.MusicEnabled)

	return g
}

func (g *Game) SetIPCCloser(c io.Closer) {
	g.ipcCloser = c
}

func (g *Game) ToggleVisibility() {
	g.settingsMutex.Lock()
	defer g.settingsMutex.Unlock()
	g.visible = !g.visible

	if g.visible {
		ebiten.SetWindowFloating(g.settings.Topmost)
	} else {
		// Oculta minimizando ou reduzindo escala temporária (no Ebiten ocultar janela é mais fácil usando window position longe ou ocultar desenho)
		// Porém podemos usar ebiten.MinimizeWindow() ou apenas desenhar nada e não processar.
		// Vamos simplesmente não desenhar o Ratinho quando oculto.
	}
}

func (g *Game) IsVisible() bool {
	g.settingsMutex.Lock()
	defer g.settingsMutex.Unlock()
	return g.visible
}

func (g *Game) onBeatDetected(intensity float64, bpm float64) {
	// Aceleração proporcional ao BPM
	g.settingsMutex.Lock()
	speedRatio := bpm / g.currentPet.BaseBPM
	if speedRatio < 0.5 {
		speedRatio = 0.5
	}
	if speedRatio > 2.5 {
		speedRatio = 2.5
	}
	g.settingsMutex.Unlock()

	// "Pulo" visual na escala
	bump := math.Min(0.25, 0.08*intensity)
	g.pulseScale = 1.0 + bump
	g.pulseDecayTimer = 7 // ~120ms a 60fps
}

func (g *Game) loadCatalog() {
	// 1. Carrega pets.json embutido
	var wrapper PetConfigWrapper
	data, err := assets.FS.ReadFile("pets.json")
	if err == nil {
		_ = json.Unmarshal(data, &wrapper)
	}

	// Cria o catálogo em memória
	for _, pc := range wrapper.Pets {
		soundVal := audio.SoundCharacter(pc.Sound)
		gifName := pc.ID + ".gif"
		if pc.ID != "rato" {
			gifName = "Novos/" + gifName
		}

		gifBytes, err := assets.FS.ReadFile(gifName)
		if err != nil {
			continue
		}

		player, err := decodeGif(gifBytes)
		if err != nil {
			continue
		}

		g.catalog = append(g.catalog, &PetDefinition{
			ID:          pc.ID,
			DisplayName: pc.DisplayName,
			BaseBPM:     pc.BaseBPM,
			Sound:       soundVal,
			GifPlayer:   player,
		})
	}

	// 2. Escaneia pasta local $HOME/.config/ratinhodesktop/Assets por GIFs adicionais
	configDir, err := os.UserConfigDir()
	if err == nil {
		assetsPath := filepath.Join(configDir, "ratinhodesktop", "Assets")
		if _, err := os.Stat(assetsPath); err == nil {
			// Lê pets.json customizado se existir
			var customMap = make(map[string]PetConfig)
			customJsonPath := filepath.Join(assetsPath, "pets.json")
			if customData, err := os.ReadFile(customJsonPath); err == nil {
				var customWrapper PetConfigWrapper
				if json.Unmarshal(customData, &customWrapper) == nil {
					for _, cp := range customWrapper.Pets {
						customMap[strings.ToLower(cp.ID)] = cp
					}
				}
			}

			// Escaneia GIFs
			_ = filepath.Walk(assetsPath, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".gif") {
					filename := strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
					id := strings.ToLower(filename)

					// Evita duplicar embutidos
					for _, pet := range g.catalog {
						if pet.ID == id {
							return nil
						}
					}

					gifBytes, err := os.ReadFile(path)
					if err != nil {
						return nil
					}

					player, err := decodeGif(gifBytes)
					if err != nil {
						return nil
					}

					displayName := formatDisplayName(filename)
					soundChar := audio.SoundPop
					if strings.Contains(id, "vaca") || strings.Contains(id, "cow") {
						soundChar = audio.SoundMoo
					} else if strings.Contains(id, "cat") || strings.Contains(id, "gato") {
						soundChar = audio.SoundMeow
					} else if strings.Contains(id, "rato") || strings.Contains(id, "mouse") {
						soundChar = audio.SoundSqueak
					} else if strings.Contains(id, "eisque") {
						soundChar = audio.SoundChime
					}

					baseBpm := 120.0

					// Sobrescreve com valores do pets.json local
					if cfg, ok := customMap[id]; ok {
						if cfg.DisplayName != "" {
							displayName = cfg.DisplayName
						}
						if cfg.BaseBPM > 0 {
							baseBpm = cfg.BaseBPM
						}
						if cfg.Sound != "" {
							soundChar = audio.SoundCharacter(cfg.Sound)
						}
					}

					g.catalog = append(g.catalog, &PetDefinition{
						ID:          id,
						DisplayName: displayName,
						BaseBPM:     baseBpm,
						Sound:       soundChar,
						GifPlayer:   player,
					})
				}
				return nil
			})
		}
	}
}

func decodeGif(data []byte) (*GifPlayer, error) {
	g, err := gif.DecodeAll(bytesNewReader(data))
	if err != nil {
		return nil, err
	}

	width, height := g.Config.Width, g.Config.Height
	if width == 0 || height == 0 {
		if len(g.Image) > 0 {
			width = g.Image[0].Bounds().Dx()
			height = g.Image[0].Bounds().Dy()
		}
	}

	player := &GifPlayer{
		Frames: make([]*ebiten.Image, len(g.Image)),
		Delays: make([]int, len(g.Delay)),
	}

	canvas := image.NewRGBA(image.Rect(0, 0, width, height))

	for i, img := range g.Image {
		draw.Draw(canvas, img.Bounds(), img, img.Bounds().Min, draw.Over)
		player.Frames[i] = ebiten.NewImageFromImage(canvas)
		player.Delays[i] = g.Delay[i]
	}

	return player, nil
}

func bytesNewReader(b []byte) io.ReadSeeker {
	return bytes.NewReader(b)
}

func formatDisplayName(s string) string {
	reg := regexp.MustCompile(`[-_]+`)
	s = reg.ReplaceAllString(s, " ")
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func (g *Game) selectPet(id string, playSound bool) {
	var target *PetDefinition
	for _, pet := range g.catalog {
		if pet.ID == id {
			target = pet
			break
		}
	}
	if target == nil && len(g.catalog) > 0 {
		target = g.catalog[0]
	}
	if target != nil {
		g.currentPet = target
		g.settings.PetID = target.ID
		if playSound {
			audio.PlayClickSound(target.Sound)
		}
	}
}

func (g *Game) saveCurrentSettings() {
	// Copia de forma segura e salva
	g.settingsMutex.Lock()
	s := g.settings
	g.settingsMutex.Unlock()
	
	// Executa em goroutine para evitar I/O bloqueante na thread principal
	go SaveSettings(s)
}

func (g *Game) setupMenu() {
	var petItems []*MenuItem
	for _, pet := range g.catalog {
		p := pet // Captura local para closure
		petItems = append(petItems, &MenuItem{
			Text:    p.DisplayName,
			Checked: p.ID == g.settings.PetID,
			Action: func() {
				g.selectPet(p.ID, true)
				g.saveCurrentSettings()
				g.setupMenu() // Reconfigura para atualizar checked
			},
		})
	}

	menuBichinho := &MenuItem{
		Text:    "Bichinho",
		Submenu: NewMenu(petItems),
	}

	menuTamanho := &MenuItem{
		Text: "Tamanho",
		Submenu: NewMenu([]*MenuItem{
			{
				Text:    "Pequeno (100px)",
				Checked: g.settings.Size == 100,
				Action: func() {
					g.resizeWindow(100)
				},
			},
			{
				Text:    "Medio (200px)",
				Checked: g.settings.Size == 200,
				Action: func() {
					g.resizeWindow(200)
				},
			},
			{
				Text:    "Grande (320px)",
				Checked: g.settings.Size == 320,
				Action: func() {
					g.resizeWindow(320)
				},
			},
		}),
	}

	menuSom := &MenuItem{
		Text: "Efeitos Sonoros",
		Submenu: NewMenu([]*MenuItem{
			{
				Text:    "Som ao Clicar",
				Checked: g.settings.SqueakEnabled,
				Action: func() {
					g.settings.SqueakEnabled = !g.settings.SqueakEnabled
					audio.SetSqueakEnabled(g.settings.SqueakEnabled)
					g.saveCurrentSettings()
					g.setupMenu()
				},
			},
			{
				Text:    "Musica de Fundo (Retro)",
				Checked: g.settings.MusicEnabled,
				Action: func() {
					g.settings.MusicEnabled = !g.settings.MusicEnabled
					audio.SetMusicEnabled(g.settings.MusicEnabled)
					g.saveCurrentSettings()
					g.setupMenu()
				},
			},
		}),
	}

	menuOpacity := &MenuItem{
		Text: "Opacidade",
		Submenu: NewMenu([]*MenuItem{
			{Text: "100%", Checked: g.settings.Opacity == 1.0, Action: func() { g.setOpacity(1.0) }},
			{Text: "75%", Checked: g.settings.Opacity == 0.75, Action: func() { g.setOpacity(0.75) }},
			{Text: "50%", Checked: g.settings.Opacity == 0.5, Action: func() { g.setOpacity(0.5) }},
			{Text: "25%", Checked: g.settings.Opacity == 0.25, Action: func() { g.setOpacity(0.25) }},
		}),
	}

	g.menu = NewMenu([]*MenuItem{
		menuBichinho,
		menuTamanho,
		{
			Text:    "Modo DVD (Quicar)",
			Checked: g.isDvdMode,
			Action: func() {
				g.toggleDvdMode()
			},
		},
		menuSom,
		{
			Text:    "Sincronizar com Musica (Beta)",
			Checked: g.settings.AudioReactiveEnabled,
			Action: func() {
				g.settings.AudioReactiveEnabled = !g.settings.AudioReactiveEnabled
				if g.settings.AudioReactiveEnabled {
					g.reactiveService.Start()
				} else {
					g.reactiveService.Stop()
					g.pulseScale = 1.0
				}
				g.saveCurrentSettings()
				g.setupMenu()
			},
		},
		menuOpacity,
		{
			Text:    "Sempre no Topo",
			Checked: g.settings.Topmost,
			Action: func() {
				g.settings.Topmost = !g.settings.Topmost
				ebiten.SetWindowFloating(g.settings.Topmost)
				g.saveCurrentSettings()
				g.setupMenu()
			},
		},
		{
			Text: "Ocultar",
			Action: func() {
				g.ToggleVisibility()
			},
		},
		{
			Text: "Sair",
			Action: func() {
				if g.ipcCloser != nil {
					_ = g.ipcCloser.Close()
				}
				g.reactiveService.Stop()
				os.Exit(0)
			},
		},
	})
}

func (g *Game) resizeWindow(newSize float64) {
	g.settings.Size = newSize
	ebiten.SetWindowSize(int(newSize)+10, int(newSize)+10)
	g.saveCurrentSettings()
	g.setupMenu()
}

func (g *Game) setOpacity(op float64) {
	g.settings.Opacity = op
	g.saveCurrentSettings()
	g.setupMenu()
}

func (g *Game) toggleDvdMode() {
	g.isDvdMode = !g.isDvdMode
	if g.isDvdMode {
		// Velocidade inicial aleatória
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		g.vx = r.Float64()*3.0 + 3.0
		g.vy = r.Float64()*3.0 + 3.0
		if r.Intn(2) == 0 {
			g.vx = -g.vx
		}
		if r.Intn(2) == 0 {
			g.vy = -g.vy
		}
	}
	g.setupMenu()
}

// Update loop do Ebiten (60 chamadas por segundo)
func (g *Game) Update() error {
	if !g.visible {
		// Se invisível, aceita atalhos internos/Update mínimo para não travar
		return nil
	}

	mx, my := ebiten.CursorPosition()
	clickLeft := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	clickRight := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)

	// Gerenciamento do Menu de Contexto
	if g.menuOpen {
		if clickLeft || clickRight {
			// Trata clique no menu
			g.menu.Update(mx, my, true)
			if !g.menu.Visible {
				g.menuOpen = false
				// Restaura tamanho padrão da janela
				ebiten.SetWindowSize(int(g.settings.Size)+10, int(g.settings.Size)+10)
			}
		} else {
			// Update hover
			g.menu.Update(mx, my, false)
		}
		return nil
	}

	// Abrir menu com botão direito
	if clickRight {
		g.menuOpen = true
		g.menu.X = int(g.settings.Size) + 5
		g.menu.Y = 5
		g.menu.Visible = true
		
		// Aumenta a largura da janela para acomodar o menu transparente
		menuHeight := len(g.menu.Items) * g.menu.ItemHeight
		winW := int(g.settings.Size) + 180 + 10
		winH := int(g.settings.Size) + 10
		if menuHeight+10 > winH {
			winH = menuHeight + 10
		}
		ebiten.SetWindowSize(winW, winH)
		return nil
	}

	// Movimentação por arraste de clique esquerdo
	if clickLeft {
		// Verifica clique duplo
		now := time.Now()
		if now.Sub(g.lastClickTime) < 300*time.Millisecond {
			g.toggleDvdMode()
			g.lastClickTime = time.Time{} // Limpa para evitar triplo clique
		} else {
			g.lastClickTime = now
			// Inicia arraste se estiver dentro do Ratinho
			petSize := int(g.settings.Size)
			if mx >= 5 && mx < petSize+5 && my >= 5 && my < petSize+5 {
				g.isDragging = true
				g.dragStartX = mx
				g.dragStartY = my
				audio.PlayClickSound(g.currentPet.Sound)
			}
		}
	}

	if g.isDragging {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			curX, curY := ebiten.CursorPosition()
			dx := curX - g.dragStartX
			dy := curY - g.dragStartY
			
			// Se moveu o suficiente, arrasta
			if math.Abs(float64(dx)) > 1 || math.Abs(float64(dy)) > 1 {
				wx, wy := ebiten.WindowPosition()
				ebiten.SetWindowPosition(wx+dx, wy+dy)
			}
		} else {
			g.isDragging = false
		}
	}

	// Lógica do Modo DVD
	if g.isDvdMode && !g.isDragging {
		wx, wy := ebiten.WindowPosition()
		sw, sh := ebiten.MonitorResolution()
		
		// Se não encontrou resolução do monitor, usa um fallback razoável
		if sw == 0 || sh == 0 {
			sw, sh = 1920, 1080
		}

		winSize := int(g.settings.Size) + 10
		wx += int(g.vx)
		wy += int(g.vy)

		bounced := false

		// Colisão Horizontal
		if wx <= 0 {
			wx = 0
			g.vx = -g.vx
			bounced = true
		} else if wx+winSize >= sw {
			wx = sw - winSize
			g.vx = -g.vx
			bounced = true
		}

		// Colisão Vertical
		if wy <= 0 {
			wy = 0
			g.vy = -g.vy
			bounced = true
		} else if wy+winSize >= sh {
			wy = sh - winSize
			g.vy = -g.vy
			bounced = true
		}

		ebiten.SetWindowPosition(wx, wy)

		// Direção horizontal determina espelhamento
		if g.vx < 0 {
			g.flipSign = -1.0
		} else {
			g.flipSign = 1.0
		}

		if bounced {
			audio.PlayClickSound(g.currentPet.Sound)
			g.flashTimer = 6 // Flash colisão (reduz opacidade por ~100ms)
		}
	} else if !g.isDragging {
		// Se não estiver no modo DVD nem arrastando, reseta espelhamento para o padrão
		g.flipSign = 1.0
	}

	// Decaimento do pulso da batida
	if g.pulseDecayTimer > 0 {
		g.pulseDecayTimer--
		if g.pulseDecayTimer == 0 {
			g.pulseScale = 1.0
		}
	}

	// Decaimento do temporizador de colisão
	if g.flashTimer > 0 {
		g.flashTimer--
	}

	// Avanço de frame do GIF do pet atual
	p := g.currentPet.GifPlayer
	if p != nil && len(p.Frames) > 0 {
		speedRatio := 1.0
		if g.settings.AudioReactiveEnabled {
			_, bpm := g.reactiveService.GetInfo()
			speedRatio = bpm / g.currentPet.BaseBPM
			if speedRatio < 0.5 {
				speedRatio = 0.5
			}
			if speedRatio > 2.5 {
				speedRatio = 2.5
			}
		}

		// 1 centisegundo = 10ms. Ebiten roda a 60fps (~16.6ms por frame)
		// Multiplicamos pela taxa de velocidade (speedRatio)
		dtCentisec := (1.0 / 60.0) * 100.0 * speedRatio
		p.Timer += dtCentisec

		frameDelay := p.Delays[p.Index]
		if frameDelay <= 0 {
			frameDelay = 10 // Padrão 100ms se inválido
		}

		if p.Timer >= float64(frameDelay) {
			p.Timer -= float64(frameDelay)
			p.Index = (p.Index + 1) % len(p.Frames)
		}
	}

	return nil
}

// Draw renderiza a tela
func (g *Game) Draw(screen *ebiten.Image) {
	if !g.visible {
		return
	}

	// Limpa tela com total transparência
	screen.Fill(color.Transparent)

	// Calcula a opacidade final considerando colisão flash
	opac := g.settings.Opacity
	if g.flashTimer > 0 {
		opac *= 0.8
	}

	p := g.currentPet.GifPlayer
	if p != nil && len(p.Frames) > 0 {
		frame := p.Frames[p.Index]
		w, h := frame.Bounds().Dx(), frame.Bounds().Dy()

		op := &ebiten.DrawImageOptions{}

		// Origem ao centro
		op.GeoM.Translate(-float64(w)/2, -float64(h)/2)

		// Escala do pet + efeito de pulso + flip do DVD
		scaleBase := g.settings.Size / float64(w)
		op.GeoM.Scale(scaleBase*g.flipSign*g.pulseScale, scaleBase*g.pulseScale)

		// Translada para a área central da janela (com 5px de padding)
		centerOffset := g.settings.Size/2 + 5
		op.GeoM.Translate(centerOffset, centerOffset)

		// Opacidade
		op.ColorScale.ScaleAlpha(float32(opac))

		screen.DrawImage(frame, op)
	}

	// Renderiza o menu se ativo
	if g.menuOpen {
		g.menu.Draw(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return outsideWidth, outsideHeight
}
