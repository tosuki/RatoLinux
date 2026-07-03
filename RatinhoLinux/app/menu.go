package app

import (
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text"
	"golang.org/x/image/font/basicfont"
)

// MenuItem representa um item no menu de contexto.
type MenuItem struct {
	Text     string
	Checked  bool
	Action   func()
	Submenu  *Menu
	IsSlider bool // Reservado para ajustes futuros se necessário
}

// Menu representa a estrutura do menu de contexto.
type Menu struct {
	Items      []*MenuItem
	X, Y       int
	Width      int
	ItemHeight int
	Selected   int
	Visible    bool
	SubActive  *Menu
	Parent     *Menu
}

// NewMenu cria uma nova instância de Menu.
func NewMenu(items []*MenuItem) *Menu {
	return &Menu{
		Items:      items,
		Width:      180,
		ItemHeight: 20,
		Selected:   -1,
	}
}

// Draw renderiza o menu de contexto na tela.
func (m *Menu) Draw(screen *ebiten.Image) {
	if !m.Visible {
		return
	}

	height := len(m.Items) * m.ItemHeight
	// Caixa de fundo com borda
	drawRect(screen, m.X, m.Y, m.Width, height, color.RGBA{30, 30, 30, 240})       // Cinza Escuro
	drawBorder(screen, m.X, m.Y, m.Width, height, color.RGBA{63, 63, 63, 255})    // Cinza Claro

	for i, item := range m.Items {
		itemY := m.Y + i*m.ItemHeight

		// Se o mouse estiver sobre o item
		if i == m.Selected {
			drawRect(screen, m.X+1, itemY+1, m.Width-2, m.ItemHeight-2, color.RGBA{0, 120, 215, 255}) // Azul Windows
		}

		// Marcador de Checked ([x] ou um marcador retro)
		checkStr := "  "
		if item.Checked {
			checkStr = "v "
		}

		// Desenhar texto
		txt := checkStr + item.Text
		if item.Submenu != nil {
			txt = txt + "  >"
		}
		
		txtColor := color.White
		if i == m.Selected {
			txtColor = color.White
		}

		// basicfont.Face7x13 tem altura de 13px, desenhamos com offset Y de ~14px
		text.Draw(screen, txt, basicfont.Face7x13, m.X+6, itemY+14, txtColor)
	}

	// Renderiza o submenu se estiver aberto
	if m.SubActive != nil && m.SubActive.Visible {
		m.SubActive.Draw(screen)
	}
}

// Update gerencia o estado e a interação do mouse com o menu.
func (m *Menu) Update(mx, my int, click bool) bool {
	if !m.Visible {
		return false
	}

	// Se houver um submenu ativo, repassa o update primeiro
	if m.SubActive != nil && m.SubActive.Visible {
		if handled := m.SubActive.Update(mx, my, click); handled {
			return true
		}
	}

	height := len(m.Items) * m.ItemHeight
	inBounds := mx >= m.X && mx < m.X+m.Width && my >= m.Y && my < m.Y+height

	if inBounds {
		m.Selected = (my - m.Y) / m.ItemHeight
		if m.Selected < 0 {
			m.Selected = 0
		}
		if m.Selected >= len(m.Items) {
			m.Selected = len(m.Items) - 1
		}

		item := m.Items[m.Selected]

		// Se tem submenu, abre ao passar o mouse
		if item.Submenu != nil {
			m.SubActive = item.Submenu
			m.SubActive.X = m.X + m.Width - 5
			m.SubActive.Y = m.Y + m.Selected*m.ItemHeight
			m.SubActive.Visible = true
			m.SubActive.Parent = m
		} else {
			// Se mover para outro item sem submenu, fecha submenu ativo
			if m.SubActive != nil {
				m.SubActive.Visible = false
				m.SubActive = nil
			}
		}

		if click {
			if item.Action != nil {
				item.Action()
				m.CloseAll()
				return true
			}
		}
		return true
	}

	// Se clicou fora, fecha o menu
	if click {
		m.CloseAll()
		return false
	}

	// Limpa seleção se o mouse sair e não houver submenu aberto para este item
	if m.SubActive == nil {
		m.Selected = -1
	}

	return false
}

// CloseAll fecha este menu e todos os submenus/parentes recursivamente.
func (m *Menu) CloseAll() {
	m.Visible = false
	m.Selected = -1
	if m.SubActive != nil {
		m.SubActive.CloseAll()
		m.SubActive = nil
	}
	curr := m
	for curr.Parent != nil {
		curr = curr.Parent
		curr.Visible = false
		curr.Selected = -1
		curr.SubActive = nil
	}
}

// Funções auxiliares de desenho básico de formas

func drawRect(dst *ebiten.Image, x, y, width, height int, clr color.Color) {
	rectImg := ebiten.NewImage(width, height)
	rectImg.Fill(clr)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(rectImg, op)
}

func drawBorder(dst *ebiten.Image, x, y, width, height int, clr color.Color) {
	// Topo e Fundo
	drawRect(dst, x, y, width, 1, clr)
	drawRect(dst, x, y+height-1, width, 1, clr)
	// Esquerda e Direita
	drawRect(dst, x, y, 1, height, clr)
	drawRect(dst, x+width-1, y, 1, height, clr)
}

// Helper para sanitizar strings de exibição (ex: substitui v por checkmark unicode se suportado,
// mas basicfont usa ASCII então 'v ' é o indicador de checked perfeito).
func checkMark() string {
	return "v "
}
