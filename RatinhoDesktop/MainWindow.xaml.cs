using System;
using System.IO;
using System.Media;
using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Interop;
using System.Windows.Media;
using System.Windows.Media.Imaging;
using System.Windows.Threading;
using RatinhoDesktop.Models;
using RatinhoDesktop.Services;
using WpfAnimatedGif;

namespace RatinhoDesktop;

public partial class MainWindow : Window
{
    private DispatcherTimer? _dvdTimer;
    private double _vx = 4.0;
    private double _vy = 4.0;
    private bool _isDvdMode = false;

    private string? _melodyPath;
    private SoundPlayer? _clickSoundPlayer;
    private MediaPlayer? _backgroundMusic;

    private bool _squeakEnabled = true;
    private bool _musicEnabled = false;
    private double _currentOpacity = 1.0;

    private System.Windows.Forms.NotifyIcon? _notifyIcon;
    private HwndSource? _source;

    // --- Seleção de bichinho ---
    private PetDefinition _currentPet = PetDefinition.Catalog[0];
    private readonly System.Collections.Generic.Dictionary<string, System.Windows.Controls.MenuItem> _petMenuItems = new();

    // --- Configurações persistidas ---
    private AppSettings _settings = new();

    // --- Transform combinado (espelhamento do modo DVD + "pulso" de batida) ---
    private readonly ScaleTransform _scaleTransform = new ScaleTransform(1.0, 1.0);
    private double _flipSign = 1.0;
    private double _pulseScale = 1.0;
    private DispatcherTimer? _pulseResetTimer;

    // --- Estimativa de BPM e arrasto ---
    private readonly System.Collections.Generic.List<DateTime> _beatTimes = new();
    private double _estimatedBpm = 120.0;
    private double _lastAppliedSpeedRatio = 1.0;
    private System.Windows.Point _dragStartPoint;

    // --- Sincronização com a música tocando no computador ---
    private AudioReactiveService? _audioReactive;

    [DllImport("user32.dll")]
    private static extern bool RegisterHotKey(IntPtr hWnd, int id, uint fsModifiers, uint vk);

    [DllImport("user32.dll")]
    private static extern bool UnregisterHotKey(IntPtr hWnd, int id);

    private const int HOTKEY_ID = 9000;
    private const uint MOD_ALT = 0x0001;
    private const uint MOD_CONTROL = 0x0002;
    private const uint VK_R = 0x52; // Key 'R'

    public MainWindow()
    {
        InitializeComponent();
    }

    private void Window_Loaded(object sender, RoutedEventArgs e)
    {
        // Carrega as preferências salvas na última execução (se existirem)
        _settings = SettingsManager.Load();

        // Aplica o transform combinado (flip + pulso) desde o início
        RatoImage.RenderTransform = _scaleTransform;

        // Monta o submenu "Bichinho" com todos os gifs disponíveis
        BuildPetMenu();

        // Seleciona o bichinho salvo (ou o padrão, o ratinho)
        SelectPet(PetDefinition.GetByIdOrDefault(_settings.PetId), playClickSound: false, persist: false);

        // Posicionar o bichinho no centro da tela inicialmente
        double screenWidth = SystemParameters.PrimaryScreenWidth;
        double screenHeight = SystemParameters.PrimaryScreenHeight;
        this.Left = (screenWidth - this.Width) / 2;
        this.Top = (screenHeight - this.Height) / 2;

        // Aplica tamanho salvo
        SetSize(_settings.Size, updateMenu: true);

        // Gerar a música de fundo sintetizada na pasta de execução local
        string assetsPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "Assets");
        _melodyPath = SoundGenerator.GenerateMelodyFile(assetsPath);

        // Inicializar MediaPlayer para a música de fundo
        if (File.Exists(_melodyPath))
        {
            _backgroundMusic = new MediaPlayer();
            _backgroundMusic.Open(new Uri(_melodyPath));
            _backgroundMusic.Volume = 0.35; // volume confortável

            // Loop da música
            _backgroundMusic.MediaEnded += (s, ev) =>
            {
                if (_musicEnabled && _backgroundMusic != null)
                {
                    _backgroundMusic.Position = TimeSpan.Zero;
                    _backgroundMusic.Play();
                }
            };
        }

        // Aplica preferências salvas de som/música/opacidade/topmost
        _squeakEnabled = _settings.SqueakEnabled;
        MenuSoundClick.IsChecked = _squeakEnabled;

        _musicEnabled = _settings.MusicEnabled;
        MenuMusic.IsChecked = _musicEnabled;
        if (_musicEnabled) _backgroundMusic?.Play();

        _currentOpacity = _settings.Opacity;
        this.Opacity = _currentOpacity;

        this.Topmost = _settings.Topmost;
        MenuTopmost.IsChecked = _settings.Topmost;

        // Configurar timer para o Modo DVD (aprox. 60 FPS)
        _dvdTimer = new DispatcherTimer
        {
            Interval = TimeSpan.FromMilliseconds(16)
        };
        _dvdTimer.Tick += DvdTimer_Tick;

        // Inicializar o ícone na bandeja do sistema (System Tray)
        InitializeTrayIcon();

        // Reativa a sincronização com música se estava ligada da última vez
        if (_settings.AudioReactiveEnabled)
        {
            MenuAudioReactive.IsChecked = true;
            StartAudioReactive();
        }
    }

    // --- Seleção de bichinho ---

    private void BuildPetMenu()
    {
        MenuBichinho.Items.Clear();
        _petMenuItems.Clear();

        foreach (var pet in PetDefinition.Catalog)
        {
            var item = new System.Windows.Controls.MenuItem
            {
                Header = pet.DisplayName,
                IsCheckable = true,
                Tag = pet.Id
            };
            item.Click += PetMenuItem_Click;
            MenuBichinho.Items.Add(item);
            _petMenuItems[pet.Id] = item;
        }
    }

    private void PetMenuItem_Click(object sender, RoutedEventArgs e)
    {
        if (sender is System.Windows.Controls.MenuItem item && item.Tag is string petId)
        {
            var pet = PetDefinition.GetByIdOrDefault(petId);
            SelectPet(pet, playClickSound: true, persist: true);
        }
    }

    private void SelectPet(PetDefinition pet, bool playClickSound, bool persist)
    {
        _currentPet = pet;

        // Troca o gif animado exibido
        var bitmap = new BitmapImage();
        bitmap.BeginInit();
        bitmap.UriSource = new Uri(pet.PackUri, UriKind.Absolute);
        bitmap.EndInit();
        ImageBehavior.SetAnimatedSource(RatoImage, bitmap);

        // Atualiza o "check" no submenu para refletir a seleção atual
        foreach (var kvp in _petMenuItems)
        {
            kvp.Value.IsChecked = kvp.Key == pet.Id;
        }

        // Gera (se necessário) e carrega o som de clique específico deste bichinho
        string assetsPath = Path.Combine(AppDomain.CurrentDomain.BaseDirectory, "Assets");
        string soundPath = SoundGenerator.GenerateCharacterSoundFile(assetsPath, pet.Sound);

        _clickSoundPlayer?.Dispose();
        _clickSoundPlayer = null;
        if (File.Exists(soundPath))
        {
            _clickSoundPlayer = new SoundPlayer(soundPath);
            _clickSoundPlayer.Load();
        }

        if (playClickSound)
        {
            PlayClickSound();
        }

        if (persist)
        {
            _settings.PetId = pet.Id;
            SettingsManager.Save(_settings);
        }

        // Força recálculo da velocidade para o novo pet
        _lastAppliedSpeedRatio = 0.0;
        AdjustAnimationSpeed();
    }

    private void RatoImage_MouseDown(object sender, MouseButtonEventArgs e)
    {
        if (e.ChangedButton == MouseButton.Left)
        {
            if (e.ClickCount == 2)
            {
                ToggleDvdMode();
                e.Handled = true;
            }
            else
            {
                _dragStartPoint = e.GetPosition(this);
            }
        }
    }

    private void RatoImage_MouseMove(object sender, System.Windows.Input.MouseEventArgs e)
    {
        if (e.LeftButton == MouseButtonState.Pressed && !_isDvdMode)
        {
            System.Windows.Point currentPoint = e.GetPosition(this);
            Vector diff = _dragStartPoint - currentPoint;

            // Só inicia o arrasto se o mouse se mover além do limite padrão do Windows
            if (Math.Abs(diff.X) > SystemParameters.MinimumHorizontalDragDistance ||
                Math.Abs(diff.Y) > SystemParameters.MinimumVerticalDragDistance)
            {
                PlayClickSound();
                try
                {
                    DragMove();
                }
                catch
                {
                    // Ignora erros se o arrasto for interrompido abruptamente
                }
            }
        }
    }

    private void PlayClickSound()
    {
        if (_squeakEnabled && _clickSoundPlayer != null)
        {
            try
            {
                _clickSoundPlayer.Play();
            }
            catch
            {
                // Ignora se der erro de concorrência ou bloqueio
            }
        }
    }

    private void ToggleDvdMode()
    {
        _isDvdMode = !_isDvdMode;
        MenuDvdMode.IsChecked = _isDvdMode;

        if (_isDvdMode)
        {
            // Começa com velocidade aleatória
            Random rand = new Random();
            _vx = (rand.Next(0, 2) == 0 ? -1 : 1) * (rand.NextDouble() * 3.0 + 3.0);
            _vy = (rand.Next(0, 2) == 0 ? -1 : 1) * (rand.NextDouble() * 3.0 + 3.0);
            _dvdTimer?.Start();
        }
        else
        {
            _dvdTimer?.Stop();
        }
    }

    private void DvdTimer_Tick(object? sender, EventArgs e)
    {
        double screenWidth = SystemParameters.PrimaryScreenWidth;
        double screenHeight = SystemParameters.PrimaryScreenHeight;

        double left = this.Left;
        double top = this.Top;
        double width = this.ActualWidth;
        double height = this.ActualHeight;

        left += _vx;
        top += _vy;

        bool bounced = false;

        // Limites horizontais
        if (left <= 0)
        {
            left = 0;
            _vx = -_vx;
            bounced = true;
        }
        else if (left + width >= screenWidth)
        {
            left = screenWidth - width;
            _vx = -_vx;
            bounced = true;
        }

        // Limites verticais
        if (top <= 0)
        {
            top = 0;
            _vy = -_vy;
            bounced = true;
        }
        else if (top + height >= screenHeight)
        {
            top = screenHeight - height;
            _vy = -_vy;
            bounced = true;
        }

        // Atualizar espelhamento horizontal com base na direção do movimento
        // Se estiver indo para a esquerda (vx < 0), espelha horizontalmente.
        _flipSign = _vx < 0 ? -1 : 1;
        ApplyTransform();

        this.Left = left;
        this.Top = top;

        if (bounced)
        {
            PlayClickSound();
            
            // Efeito visual de colisão (mudar levemente a opacidade temporariamente)
            this.Opacity = _currentOpacity * 0.8;
            DispatcherTimer opacityRestoreTimer = new DispatcherTimer
            {
                Interval = TimeSpan.FromMilliseconds(100)
            };
            opacityRestoreTimer.Tick += (s, ev) =>
            {
                this.Opacity = _currentOpacity;
                opacityRestoreTimer.Stop();
            };
            opacityRestoreTimer.Start();
        }
    }

    // --- Sincronização com o ritmo da música ---

    private void AudioReactive_Click(object sender, RoutedEventArgs e)
    {
        if (MenuAudioReactive.IsChecked)
        {
            StartAudioReactive();
        }
        else
        {
            StopAudioReactive();
        }

        _settings.AudioReactiveEnabled = MenuAudioReactive.IsChecked;
        SettingsManager.Save(_settings);
    }

    private void StartAudioReactive()
    {
        _audioReactive ??= new AudioReactiveService();
        _audioReactive.BeatDetected -= OnBeatDetectedFromAudioThread;
        _audioReactive.BeatDetected += OnBeatDetectedFromAudioThread;

        bool started = _audioReactive.Start();
        if (!started)
        {
            MenuAudioReactive.IsChecked = false;
            System.Windows.MessageBox.Show(
                "Não foi possível capturar o áudio do sistema. Verifique se há algum " +
                "dispositivo de saída de áudio padrão configurado no Windows.",
                "Ratinho Desktop",
                MessageBoxButton.OK,
                MessageBoxImage.Warning);
        }
    }

    private void StopAudioReactive()
    {
        _audioReactive?.Stop();
        ResetPulse();
    }

    // Este evento chega em uma thread de captura de áudio (não é a thread da UI),
    // então precisamos despachar para a Dispatcher antes de mexer em qualquer controle.
    private void OnBeatDetectedFromAudioThread()
    {
        Dispatcher.BeginInvoke(new Action(PulseOnBeat));
    }

    private void UpdateBpmEstimation()
    {
        var now = DateTime.UtcNow;

        // Se a última batida foi há mais de 3 segundos, limpa o histórico
        if (_beatTimes.Count > 0 && (now - _beatTimes[^1]).TotalSeconds > 3.0)
        {
            _beatTimes.Clear();
        }

        _beatTimes.Add(now);

        if (_beatTimes.Count > 8)
        {
            _beatTimes.RemoveAt(0);
        }

        if (_beatTimes.Count >= 2)
        {
            double totalIntervalMs = 0;
            int count = 0;
            for (int i = 1; i < _beatTimes.Count; i++)
            {
                double interval = (_beatTimes[i] - _beatTimes[i - 1]).TotalMilliseconds;
                // Considera apenas intervalos que equivalem de 40 a 240 BPM (1500ms a 250ms)
                if (interval >= 250 && interval <= 1500)
                {
                    totalIntervalMs += interval;
                    count++;
                }
            }

            if (count > 0)
            {
                double avgIntervalMs = totalIntervalMs / count;
                _estimatedBpm = 60000.0 / avgIntervalMs;
            }
        }
        else
        {
            _estimatedBpm = 120.0;
        }
    }

    private void AdjustAnimationSpeed()
    {
        if (MenuAudioReactive.IsChecked)
        {
            double targetRatio = _estimatedBpm / _currentPet.BaseBpm;
            targetRatio = Math.Clamp(targetRatio, 0.5, 2.5);

            // Apenas aplica a alteração de velocidade se houver mudança relevante para evitar engasgos
            if (Math.Abs(targetRatio - _lastAppliedSpeedRatio) > 0.05)
            {
                _lastAppliedSpeedRatio = targetRatio;
                ImageBehavior.SetAnimationSpeedRatio(RatoImage, targetRatio);
            }
        }
        else
        {
            if (_lastAppliedSpeedRatio != 1.0)
            {
                _lastAppliedSpeedRatio = 1.0;
                ImageBehavior.SetAnimationSpeedRatio(RatoImage, 1.0);
            }
        }
    }

    private void PulseOnBeat()
    {
        UpdateBpmEstimation();
        AdjustAnimationSpeed();

        double intensity = _audioReactive?.LastBeatIntensity ?? 1.0;

        // "Pulo" visual proporcional à força da batida
        double bump = Math.Min(0.25, 0.08 * intensity);
        _pulseScale = 1.0 + bump;
        ApplyTransform();

        _pulseResetTimer?.Stop();
        _pulseResetTimer = new DispatcherTimer { Interval = TimeSpan.FromMilliseconds(120) };
        _pulseResetTimer.Tick += (s, ev) =>
        {
            _pulseScale = 1.0;
            ApplyTransform();
            _pulseResetTimer?.Stop();
        };
        _pulseResetTimer.Start();
    }

    private void ResetPulse()
    {
        _pulseScale = 1.0;
        ApplyTransform();
        
        _beatTimes.Clear();
        _estimatedBpm = 120.0;
        AdjustAnimationSpeed();
    }

    private void ApplyTransform()
    {
        _scaleTransform.ScaleX = _flipSign * _pulseScale;
        _scaleTransform.ScaleY = _pulseScale;
    }

    // --- Ações do Menu de Contexto ---

    private void Size_Pequeno_Click(object sender, RoutedEventArgs e)
    {
        SetSize(100, updateMenu: true);
        _settings.Size = 100;
        SettingsManager.Save(_settings);
    }

    private void Size_Medio_Click(object sender, RoutedEventArgs e)
    {
        SetSize(200, updateMenu: true);
        _settings.Size = 200;
        SettingsManager.Save(_settings);
    }

    private void Size_Grande_Click(object sender, RoutedEventArgs e)
    {
        SetSize(320, updateMenu: true);
        _settings.Size = 320;
        SettingsManager.Save(_settings);
    }

    private void SetSize(double size, bool updateMenu)
    {
        RatoImage.Width = size;
        RatoImage.Height = size;
        this.Width = size + 10; // adiciona padding
        this.Height = size + 10;

        if (updateMenu)
        {
            System.Windows.Controls.MenuItem target = size switch
            {
                <= 100 => MenuSizeSmall,
                >= 320 => MenuSizeLarge,
                _ => MenuSizeMedium
            };
            UpdateSizeMenuChecked(target);
        }
    }

    private void UpdateSizeMenuChecked(System.Windows.Controls.MenuItem checkedItem)
    {
        MenuSizeSmall.IsChecked = false;
        MenuSizeMedium.IsChecked = false;
        MenuSizeLarge.IsChecked = false;
        checkedItem.IsChecked = true;
    }

    private void DvdMode_Click(object sender, RoutedEventArgs e)
    {
        ToggleDvdMode();
    }

    private void SoundClick_Click(object sender, RoutedEventArgs e)
    {
        _squeakEnabled = MenuSoundClick.IsChecked;
        _settings.SqueakEnabled = _squeakEnabled;
        SettingsManager.Save(_settings);
    }

    private void Music_Click(object sender, RoutedEventArgs e)
    {
        _musicEnabled = MenuMusic.IsChecked;
        if (_backgroundMusic != null)
        {
            if (_musicEnabled)
            {
                _backgroundMusic.Position = TimeSpan.Zero;
                _backgroundMusic.Play();
            }
            else
            {
                _backgroundMusic.Pause();
            }
        }

        _settings.MusicEnabled = _musicEnabled;
        SettingsManager.Save(_settings);
    }

    private void Opacity_Click(object sender, RoutedEventArgs e)
    {
        if (sender is System.Windows.Controls.MenuItem item && double.TryParse(item.Tag as string, out double val))
        {
            _currentOpacity = val;
            this.Opacity = val;

            _settings.Opacity = val;
            SettingsManager.Save(_settings);
        }
    }

    private void Topmost_Click(object sender, RoutedEventArgs e)
    {
        this.Topmost = MenuTopmost.IsChecked;
        _settings.Topmost = this.Topmost;
        SettingsManager.Save(_settings);
    }

    private void Sair_Click(object sender, RoutedEventArgs e)
    {
        Close();
    }

    private void Ocultar_Click(object sender, RoutedEventArgs e)
    {
        this.Hide();
    }

    private void InitializeTrayIcon()
    {
        _notifyIcon = new System.Windows.Forms.NotifyIcon();
        try
        {
            string exePath = System.Diagnostics.Process.GetCurrentProcess().MainModule?.FileName ?? "";
            if (!string.IsNullOrEmpty(exePath) && File.Exists(exePath))
            {
                _notifyIcon.Icon = System.Drawing.Icon.ExtractAssociatedIcon(exePath);
            }
            else
            {
                _notifyIcon.Icon = System.Drawing.SystemIcons.Application;
            }
        }
        catch
        {
            _notifyIcon.Icon = System.Drawing.SystemIcons.Application;
        }

        _notifyIcon.Text = "Ratinho Desktop";
        _notifyIcon.Visible = true;

        // Double click on tray icon toggles window visibility
        _notifyIcon.DoubleClick += (s, e) => ToggleWindowVisibility();

        // Context menu for the tray icon
        var contextMenu = new System.Windows.Forms.ContextMenuStrip();
        
        var showHideItem = new System.Windows.Forms.ToolStripMenuItem("Mostrar / Ocultar");
        showHideItem.Click += (s, e) => ToggleWindowVisibility();
        contextMenu.Items.Add(showHideItem);
        
        contextMenu.Items.Add(new System.Windows.Forms.ToolStripSeparator());
        
        var exitItem = new System.Windows.Forms.ToolStripMenuItem("Sair");
        exitItem.Click += (s, e) => Close();
        contextMenu.Items.Add(exitItem);

        _notifyIcon.ContextMenuStrip = contextMenu;
    }

    private void ToggleWindowVisibility()
    {
        if (this.Visibility == Visibility.Visible)
        {
            this.Hide();
        }
        else
        {
            this.Show();
            if (this.WindowState == WindowState.Minimized)
            {
                this.WindowState = WindowState.Normal;
            }
            this.Activate();
        }
    }

    protected override void OnSourceInitialized(EventArgs e)
    {
        base.OnSourceInitialized(e);

        var helper = new WindowInteropHelper(this);
        _source = HwndSource.FromHwnd(helper.Handle);
        if (_source != null)
        {
            _source.AddHook(HwndHook);
            RegisterHotKey(helper.Handle, HOTKEY_ID, MOD_CONTROL | MOD_ALT, VK_R);
        }
    }

    private IntPtr HwndHook(IntPtr hwnd, int msg, IntPtr wParam, IntPtr lParam, ref bool handled)
    {
        const int WM_HOTKEY = 0x0312;
        if (msg == WM_HOTKEY && wParam.ToInt32() == HOTKEY_ID)
        {
            ToggleWindowVisibility();
            handled = true;
        }
        return IntPtr.Zero;
    }

    protected override void OnClosed(EventArgs e)
    {
        // Stop music
        if (_backgroundMusic != null)
        {
            _backgroundMusic.Stop();
            _backgroundMusic.Close();
        }

        // Stop audio-reactive capture
        _audioReactive?.Dispose();
        _audioReactive = null;

        // Unregister global hotkey
        if (_source != null)
        {
            _source.RemoveHook(HwndHook);
            var helper = new WindowInteropHelper(this);
            UnregisterHotKey(helper.Handle, HOTKEY_ID);
        }

        // Dispose system tray icon
        if (_notifyIcon != null)
        {
            _notifyIcon.Visible = false;
            _notifyIcon.Dispose();
        }

        base.OnClosed(e);
    }
}
