# Pull Request: Portabilidade do RatinhoDesktop para Linux usando Golang e Ebitengine

## 📝 Descrição
Este Pull Request introduz o port oficial do virtual pet **RatinhoDesktop** para o sistema operacional **Linux**. A aplicação foi totalmente reescrita em **Golang** utilizando a engine de jogos **Ebitengine (v2)**, garantindo fidelidade de design, baixo consumo de recursos (~10-15MB de RAM) e facilidade de compilação sem depender de runtimes pesados ou CGo complexo no host.

---

## 🚀 Funcionalidades Portadas & Implementadas

1.  **Fidelidade Visual (Ebitengine v2)**:
    *   Janela transparente sem bordas (`ebiten.SetScreenTransparent(true)` e `ebiten.SetWindowDecorated(false)`).
    *   Desenho e redimensionamento dinâmicos do mascote de acordo com as preferências (Pequeno: 100px, Médio: 200px, Grande: 320px).
    *   Movimentação arrastando o pet com o botão esquerdo e clique duplo para alternar o **Modo DVD (quicar)**.
    *   Espelhamento horizontal automático com base no vetor de movimento horizontal.

2.  **Áudio Sintetizado Localmente**:
    *   Portabilidade de 100% das fórmulas matemáticas do sintetizador original (C#) para Go, gerando os timbres chiptune retro (Squeak, Moo, Meow, Pop, Chime) e a melodia retro em loop na memória.
    *   Reprodutor de som simplificado sem cache no disco, utilizando a biblioteca de áudio nativa do Ebiten.

3.  **Áudio Reativo (Beat Detection)**:
    *   Captura dinâmica do som do sistema lendo o stream PCM do PulseAudio ou PipeWire em tempo real através do stdout de subprocessos (`parec`/`pw-record`). Isso evita a necessidade de headers CGo complexos na compilação.
    *   Algoritmo de análise de energia RMS e média móvel para detecção de picos de batida, aplicando pulsação física de escala e aceleração da taxa de quadros (FPS) do GIF proporcionalmente ao BPM estimado.

4.  **Menu de Contexto Pixel-Art**:
    *   Um menu retro em pixel-art desenhado inteiramente dentro da janela do Ebitengine (botão direito).
    *   A janela expande sua largura e altura dinamicamente para acomodar o menu principal e submenus abertos sem corte (clipping) do gerenciador de janelas do Linux, encolhendo ao seu tamanho original após o fechamento.

5.  **Instância Única e Atalhos de Teclado Globais**:
    *   Servidor local Unix Socket (`/tmp/ratinhodesktop.sock`) para controle de processo único.
    *   Suporte a argumento de CLI `--toggle` para alternar a visibilidade da janela. Isso permite aos usuários registrar atalhos de teclado globais (ex: `Ctrl+Alt+R`) diretamente nas configurações de atalhos de seus ambientes desktop (GNOME/KDE) tanto em **Wayland** quanto em **X11**.

6.  **Persistência**:
    *   Preferências salvas no formato JSON seguindo a especificação XDG em: `~/.config/ratinhodesktop/settings.json`.

---

## 🛠️ Como Testar

1.  Navegue até a pasta do port no Linux:
    ```bash
    cd RatinhoLinux
    ```
2.  Instale as dependências gráficas e de som do sistema operacional (Fedora) e compile/instale executando:
    ```bash
    chmod +x install.sh
    ./install.sh
    ```
3.  Você pode abrir o mascote diretamente pelo menu de aplicativos como **Ratinho Desktop** ou pelo terminal chamando:
    ```bash
    ratinho-desktop
    ```
4.  Para testar o controle de atalho global, tente executar `ratinho-desktop --toggle` com o mascote aberto para ver se ele oculta/exibe instantaneamente.
