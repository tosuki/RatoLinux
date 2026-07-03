# Ratinho Desktop Linux 🐭🐧

Este é o port oficial em **Golang** do **Ratinho Desktop**, um mascote virtual retro interativo e reativo que flutua na tela do seu computador. Esta versão foi desenvolvida sob a engine de jogos **Ebitengine (v2)**, garantindo uma execução leve (cerca de 10-15MB de consumo de RAM) e compatibilidade com distribuições Linux modernas.

---

## 🚀 Funcionalidades Portadas

- **Mascote Transparente**: Janela sem bordas e fundo transparente flutuando no seu desktop.
- **Movimentação por Clique**: Arrasta o Ratinho segurando o botão esquerdo do mouse.
- **Modo DVD (Quicar)**: Clique duplo para fazê-lo rebater nas bordas da tela. Ele inverte a direção horizontal de acordo com o movimento!
- **Áudio Sintetizado Localmente**: Efeitos de cliques (Squeak, Moo, Meow, Pop, Chime) e a melodia chiptune retro em loop, gerados dinamicamente via PCM na memória (sem arquivos externos obrigatórios).
- **Áudio Reativo a Músicas (Beat Detection)**: Capta a saída de som padrão do sistema (PulseAudio/PipeWire) em tempo real, estima o BPM da música e faz o Ratinho pulsar fisicamente e acelerar a animação no ritmo da batida.
- **Menu Retro Customizado**: Clique com o botão direito sobre o pet para abrir o menu pixel-art integrado, onde você pode configurar:
  - Seleção de Bichinho (com suporte a escaneamento de novos arquivos).
  - Tamanho (100px, 200px, 320px).
  - Ligar/Desligar sons e música.
  - Opacidade (100%, 75%, 50%, 25%).
  - Ativar/Desativar modo sempre no topo.
  - Ocultar e Sair.
- **Instância Única & Atalho Global**: Suporte a atalho global via linha de comando (`--toggle`) compatível com sessões Wayland e X11.

---

## 🛠️ Requisitos de Compilação

Para compilar o aplicativo, você precisa dos pacotes de desenvolvimento do X11/OpenGL e do compilador C (`gcc`) instalados no sistema, já que a engine utiliza CGO para se comunicar com o servidor gráfico.

### No Fedora:
```bash
sudo dnf install -y gcc pkgconf-pkg-config alsa-lib-devel mesa-libGL-devel libX11-devel libXrandr-devel libXcursor-devel libXinerama-devel libXi-devel libXxf86vm-devel pulseaudio-utils pipewire-utils
```

### No Ubuntu / Debian / Mint:
```bash
sudo apt install -y gcc pkg-config libx11-dev libxrandr-dev libxcursor-dev libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev pulseaudio-utils
```

---

## 📥 Como Instalar e Rodar

Você pode usar o script de instalação automática `install.sh` disponível neste diretório:

```bash
chmod +x install.sh
./install.sh
```

O script cuidará de:
1. Instalar as dependências de sistema (caso utilize Fedora).
2. Compilar o executável em Go.
3. Copiar o binário para `~/.local/bin/ratinho-desktop`.
4. Copiar os Assets para a pasta de configuração em `~/.config/ratinhodesktop/Assets`.
5. Criar o atalho no menu de aplicativos (`.desktop`).

---

## 🎹 Configurando o Atalho de Teclado Global (`Ctrl + Alt + R`)

Por motivos de segurança, ambientes desktop modernos sob **Wayland** impedem que aplicativos monitorem as teclas do teclado globalmente em background. 

Para contornar isso, o Ratinho suporta a flag `--toggle`. Ao executar o comando, se ele já estiver aberto, ele oculta ou exibe a janela. 

Para registrar o atalho:
1. Abra as **Configurações** do seu sistema (GNOME Settings, KDE System Settings, etc.).
2. Vá em **Teclado** -> **Atalhos de Teclado** -> **Atalhos Personalizados**.
3. Adicione um novo atalho:
   - **Nome**: `Alternar Ratinho`
   - **Comando**: `/home/SEU_USUARIO/.local/bin/ratinho-desktop --toggle` *(substitua pelo seu usuário ou caminho correto)*
   - **Atalho**: Pressione `Ctrl + Alt + R`
4. Salve. Pronto! Agora pressionando o atalho você oculta ou exibe o mascote.

---

## ⚙️ Arquivos de Configuração e Assets Customizados

- **Configurações**: Salvas automaticamente em formato JSON no caminho:
  `~/.config/ratinhodesktop/settings.json`
- **Pets Personalizados**: Você pode arrastar novos arquivos `.gif` para a pasta:
  `~/.config/ratinhodesktop/Assets/`
  Eles serão automaticamente identificados pelo aplicativo na próxima execução! Se quiser personalizar o som e o BPM base do novo pet, adicione a entrada correspondente no arquivo `pets.json` dentro desse mesmo diretório:

```json
{
  "pets": [
    {
      "id": "nome-do-seu-gif-em-minusculo",
      "displayName": "Nome Bonito de Exibição",
      "baseBpm": 120.0,
      "sound": "Pop" 
    }
  ]
}
```
*Sons válidos: `Squeak`, `Moo`, `Meow`, `Pop`, `Chime`.*
