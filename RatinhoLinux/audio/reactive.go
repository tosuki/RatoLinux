package audio

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// AudioReactiveService captura áudio do monitor do sistema e realiza detecção de batidas.
type AudioReactiveService struct {
	mu                   sync.RWMutex
	cmd                  *exec.Cmd
	running              bool
	lastBeatTime         time.Time
	runningAverageEnergy float64
	lastBeatIntensity    float64
	estimatedBpm         float64
	beatTimes            []time.Time
	beatCallback         func(intensity float64, bpm float64)
}

// NewAudioReactiveService inicializa a estrutura do serviço de áudio reativo.
func NewAudioReactiveService(callback func(intensity float64, bpm float64)) *AudioReactiveService {
	return &AudioReactiveService{
		lastBeatTime: time.Time{},
		estimatedBpm: 120.0,
		beatCallback: callback,
	}
}

// GetDefaultMonitorSource tenta obter o nome da fonte monitor padrão do PulseAudio/PipeWire.
func GetDefaultMonitorSource() (string, error) {
	// 1. Tenta pactl get-default-sink
	cmd := exec.Command("pactl", "get-default-sink")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err == nil {
		sink := strings.TrimSpace(stdout.String())
		if sink != "" {
			return sink + ".monitor", nil
		}
	}

	// 2. Se falhar, tenta pactl info e busca "Default Sink"
	cmdInfo := exec.Command("pactl", "info")
	stdout.Reset()
	cmdInfo.Stdout = &stdout
	if err := cmdInfo.Run(); err == nil {
		scanner := bufio.NewScanner(&stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "Default Sink:") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					sink := strings.TrimSpace(parts[1])
					if sink != "" {
						return sink + ".monitor", nil
					}
				}
			}
		}
	}

	// 3. Fallback ou erro
	return "", fmt.Errorf("não foi possível identificar o dispositivo de som monitor padrão")
}

// Start inicia a captura de áudio em background.
func (s *AudioReactiveService) Start() bool {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return true
	}
	s.mu.Unlock()

	monitorSource, err := GetDefaultMonitorSource()
	if err != nil {
		// Se falhar ao buscar o monitor, o parec pode tentar usar o padrão de microfone,
		// mas para loopback de saída, idealmente precisamos do monitor.
		// Vamos tentar rodar o parec mesmo assim sem especificar fonte como fallback.
		monitorSource = ""
	}

	args := []string{"--format=s16le", "--channels=1", "--rate=44100"}
	if monitorSource != "" {
		args = append([]string{"-d", monitorSource}, args...)
	}

	cmd := exec.Command("parec", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return false
	}

	if err := cmd.Start(); err != nil {
		// Tenta pw-record caso parec não exista
		pwArgs := []string{"--format=s16", "--channels=1", "--rate=44100"}
		if monitorSource != "" {
			pwArgs = append([]string{"--target", monitorSource}, pwArgs...)
		}
		cmd = exec.Command("pw-record", pwArgs...)
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return false
		}
		if err := cmd.Start(); err != nil {
			return false
		}
	}

	s.mu.Lock()
	s.cmd = cmd
	s.running = true
	s.runningAverageEnergy = 0.0
	s.lastBeatTime = time.Now()
	s.beatTimes = nil
	s.mu.Unlock()

	// Inicia goroutine de processamento de áudio
	go s.processAudio(stdout)

	return true
}

// Stop finaliza a captura.
func (s *AudioReactiveService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
	}
	s.running = false
	s.cmd = nil
}

// IsRunning indica se a captura está ativa.
func (s *AudioReactiveService) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

func (s *AudioReactiveService) processAudio(reader io.Reader) {

	// Buffer de leitura (ex: chunk de 1024 amostras, cada amostra = 2 bytes s16le)
	const samplesPerChunk = 1024
	const bytesPerSample = 2
	buffer := make([]byte, samplesPerChunk*bytesPerSample)

	minBeatInterval := 180 * time.Millisecond

	for {
		s.mu.RLock()
		active := s.running
		s.mu.RUnlock()
		if !active {
			break
		}

		// Lê o chunk completo do pipe
		n, err := io.ReadFull(reader, buffer)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			break
		}
		if n < bytesPerSample {
			break
		}

		numSamplesRead := n / bytesPerSample
		var sumSquares float64

		for i := 0; i < numSamplesRead; i++ {
			offset := i * bytesPerSample
			sampleVal := int16(binary.LittleEndian.Uint16(buffer[offset : offset+2]))
			// Normaliza para [-1.0, 1.0]
			normalized := float64(sampleVal) / 32768.0
			sumSquares += normalized * normalized
		}

		rms := math.Sqrt(sumSquares / float64(numSamplesRead))

		s.mu.Lock()
		// Média móvel do volume
		const smoothing = 0.05
		if s.runningAverageEnergy <= 0 {
			s.runningAverageEnergy = rms
		} else {
			s.runningAverageEnergy = (smoothing * rms) + ((1.0 - smoothing) * s.runningAverageEnergy)
		}

		const minAudibleRms = 0.02
		const beatThresholdRatio = 1.35

		loudEnough := rms > minAudibleRms
		isPeak := rms > s.runningAverageEnergy*beatThresholdRatio
		intervalOk := time.Since(s.lastBeatTime) >= minBeatInterval

		if loudEnough && isPeak && intervalOk {
			s.lastBeatTime = time.Now()
			s.lastBeatIntensity = math.Min(2.0, rms/math.Max(0.0001, s.runningAverageEnergy))
			s.updateBpmEstimation()

			// Dispara o callback na thread principal ou em goroutine
			if s.beatCallback != nil {
				go s.beatCallback(s.lastBeatIntensity, s.estimatedBpm)
			}
		}
		s.mu.Unlock()
	}
}



func (s *AudioReactiveService) updateBpmEstimation() {
	now := time.Now()

	// Se a última batida foi há mais de 3 segundos, limpa o histórico
	if len(s.beatTimes) > 0 && now.Sub(s.beatTimes[len(s.beatTimes)-1]).Seconds() > 3.0 {
		s.beatTimes = nil
	}

	s.beatTimes = append(s.beatTimes, now)

	// Mantém no máximo as últimas 8 batidas para média
	if len(s.beatTimes) > 8 {
		s.beatTimes = s.beatTimes[1:]
	}

	if len(s.beatTimes) >= 2 {
		var totalIntervalMs float64
		count := 0
		for i := 1; i < len(s.beatTimes); i++ {
			interval := float64(s.beatTimes[i].Sub(s.beatTimes[i-1]).Milliseconds())
			// Filtro: BPM de 40 a 240 (intervalos de 250ms a 1500ms)
			if interval >= 250 && interval <= 1500 {
				totalIntervalMs += interval
				count++
			}
		}

		if count > 0 {
			avgIntervalMs := totalIntervalMs / float64(count)
			s.estimatedBpm = 60000.0 / avgIntervalMs
		}
	} else {
		s.estimatedBpm = 120.0
	}
}

// GetInfo retorna a intensidade e o BPM estimado atuais de forma segura para concorrência.
func (s *AudioReactiveService) GetInfo() (float64, float64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastBeatIntensity, s.estimatedBpm
}
