#!/bin/bash
# install.sh
set -e

echo "============================================="
# Detectar distribuição e instalar dependências de desenvolvimento para Ebiten no Fedora
if [ -f /etc/os-release ]; then
    . /etc/os-release
    if [ "$ID" = "fedora" ]; then
        echo "=== 1. Instalando dependências de build (Ebitengine) no Fedora ==="
        sudo dnf install -y \
          gcc \
          pkgconf-pkg-config \
          alsa-lib-devel \
          mesa-libGL-devel \
          libX11-devel \
          libXrandr-devel \
          libXcursor-devel \
          libXinerama-devel \
          libXi-devel \
          libXxf86vm-devel \
          pulseaudio-utils \
          pipewire-utils
    else
        echo "=== Aviso: Distribuição detectada ($ID) não é Fedora. Certifique-se de instalar as dependências equivalentes manualmente. ==="
    fi
fi

echo "=== 2. Compilando o RatinhoDesktop para Linux ==="
# Compilar binário Go
go build -o ratinho-desktop

echo "=== 3. Instalando o aplicativo localmente ==="
# Criar diretórios locais sob a estrutura do usuário
mkdir -p "$HOME/.local/bin"
mkdir -p "$HOME/.config/ratinhodesktop/Assets"

# Copiar executável
cp ratinho-desktop "$HOME/.local/bin/"

# Copiar assets
cp -r assets/* "$HOME/.config/ratinhodesktop/Assets/"

echo "=== 4. Criando atalho no menu de aplicativos (.desktop) ==="
mkdir -p "$HOME/.local/share/applications"
cat <<EOF > "$HOME/.local/share/applications/ratinho-desktop.desktop"
[Desktop Entry]
Type=Application
Name=Ratinho Desktop
Comment=Mascote virtual retro interativo e reativo
Exec=$HOME/.local/bin/ratinho-desktop
Path=$HOME/.config/ratinhodesktop
Icon=system-run
Terminal=false
Categories=Game;Utility;
EOF

chmod +x "$HOME/.local/share/applications/ratinho-desktop.desktop"

echo "============================================="
echo " INSTALAÇÃO CONCLUÍDA COM SUCESSO!"
echo "============================================="
echo "Você pode iniciar o mascote executando:"
echo "  ratinho-desktop (caso ~/.local/bin esteja no seu PATH)"
echo "ou executando diretamente no terminal:"
echo "  $HOME/.local/bin/ratinho-desktop"
echo "Ou simplesmente encontrando 'Ratinho Desktop' no menu de aplicativos do seu sistema."
echo "============================================="
