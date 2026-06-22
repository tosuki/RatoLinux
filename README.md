# Ratinho Desktop 🐭

Um mascote de desktop interativo e divertido construído em C# e WPF (.NET 10). O aplicativo exibe o famoso meme do ratinho girando flutuando sobre a sua tela com fundo transparente, suporte a arrasto por clique, modo automático "DVD Screensaver" e efeitos sonoros gerados por código (chiptune).

---

## 🚀 Funcionalidades

- **Mascote Flutuante**: Janela sem bordas e fundo transparente.
- **Movimentação por Clique**: Arrasta o Ratinho para qualquer canto da sua tela segurando-o com o botão esquerdo.
- **Modo DVD (Quicar)**: Com um clique duplo com o botão esquerdo, o Ratinho começa a quicar pelas bordas da tela. O Ratinho se inverte horizontalmente de acordo com a direção do movimento!
- **Áudio Sintetizado Localmente**: Efeitos sonoros de clique e música retro de fundo criados programaticamente em tempo de compilação (sem necessidade de conexões externas).
- **Menu de Contexto Completo (Botão Direito)**:
  - **Tamanho**: Altera o tamanho entre Pequeno (100px), Médio (200px) e Grande (320px).
  - **Opacidade**: Controla a transparência da janela (100%, 75%, 50%, 25%).
  - **Sons e Música**: Configura separadamente o guincho de clique ou a música retro em loop.
  - **Sempre no Topo**: Fixa o ratinho acima de outras janelas.
  - **Sair**: Fecha o aplicativo.

---

## 🛠️ Como Executar em Modo de Desenvolvimento

Certifique-se de ter o SDK do .NET 10.0 instalado.

1. Navegue até a pasta do projeto:
   ```bash
   cd RatinhoDesktop
   ```
2. Restaure e execute o projeto:
   ```bash
   dotnet run
   ```

---

## 📦 Como Compilar e Distribuir o Executável (`.exe`)

Como se trata de uma aplicação desktop Windows, a execução é feita abrindo diretamente o arquivo `.exe`. Para distribuir o Ratinho para outras máquinas, você tem duas opções de compilação:

### Opção 1: Executável Autônomo (Self-Contained) - *Recomendado para outras máquinas*
Esta opção embute o runtime do .NET 10 dentro do próprio executável. O arquivo gerado é maior (cerca de 50MB a 60MB), mas **roda em qualquer computador Windows (x64) imediatamente ao dar duplo clique**, sem precisar que o usuário instale mais nada.

Execute o comando abaixo dentro da pasta `RatinhoDesktop`:
```bash
dotnet publish -c Release -r win-x64 --self-contained true -p:PublishSingleFile=true -p:PublishReadyToRun=true
```
O executável único compilado ficará em:
`RatinhoDesktop/bin/Release/net10.0-windows/win-x64/publish/RatinhoDesktop.exe`

### Opção 2: Executável Dependente de Framework (Lightweight)
Esta opção gera um executável super leve (apenas alguns kilobytes/megabytes), mas **exige que a máquina de destino tenha o .NET Desktop Runtime 10.0 instalado**. Se o usuário não tiver, o Windows abrirá uma janela solicitando o download.

Execute o comando abaixo dentro da pasta `RatinhoDesktop`:
```bash
dotnet publish -c Release -r win-x64 --self-contained false -p:PublishSingleFile=true
```
O executável único compilado ficará em:
`RatinhoDesktop/bin/Release/net10.0-windows/win-x64/publish/RatinhoDesktop.exe`

*Nota: Ao executar em uma nova pasta, o aplicativo gera automaticamente os arquivos de som padrão dentro de uma subpasta `Assets/`. Certifique-se de que a aplicação tenha permissão de escrita no diretório onde está sendo executada (evite executar diretamente de pastas restritas como `C:\Program Files` sem privilégios de administrador).*

---

## ⚙️ Como Configurar Atalho e Inicialização no Windows

Como o Ratinho Desktop é um aplicativo portátil, você mesmo pode configurar a inicialização automática e atalhos rápidos de teclado:

### 🎹 Criar um Atalho de Teclado Global (`Ctrl + Alt + R`)
Para poder abrir o Ratinho a qualquer momento pressionando um atalho do teclado:
1. Clique com o botão direito sobre o executável `RatinhoDesktop.exe` (gerado na pasta de publicação ou release) e escolha **Criar atalho**.
2. Clique com o botão direito no arquivo de atalho criado e clique em **Propriedades**.
3. Na aba **Atalho**, clique no campo **Tecla de atalho**.
4. Pressione a combinação de teclas desejada (ex: `Ctrl + Alt + R`).
5. Clique em **Aplicar** e **OK**.
6. Agora, basta pressionar essa combinação em qualquer lugar do Windows para abrir o app!

### 🚀 Iniciar Junto com o Windows (Auto-start)
Caso queira que o Ratinho inicie automaticamente sempre que você ligar o computador:
1. Pressione as teclas `Win + R` para abrir a janela "Executar".
2. Digite `shell:startup` e pressione Enter. Isso abrirá a pasta de inicialização do Windows.
3. Copie o atalho do `RatinhoDesktop.exe` (criado no passo anterior) e cole-o dentro dessa pasta.

---

## 📂 Estrutura do Projeto

```
Rato/
├── README.md                      # Este arquivo
└── RatinhoDesktop/
    ├── RatinhoDesktop.csproj      # Configuração do projeto e dependências NuGet
    ├── App.xaml / App.xaml.cs     # Inicialização da aplicação WPF
    ├── MainWindow.xaml            # Design XAML da janela transparente e menus
    ├── MainWindow.xaml.cs         # Lógica em C# (Modo DVD, áudios e interações)
    ├── SoundGenerator.cs          # Gerador em C# que sintetiza as ondas de áudio (.wav)
    └── Assets/
        └── rato.gif               # GIF do ratinho em rotação (recurso embutido)
```
