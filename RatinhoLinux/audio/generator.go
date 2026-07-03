package audio

import (
	"bytes"
	"encoding/binary"
	"math"
)

// SoundCharacter representa o tipo de som associado a cada mascote.
type SoundCharacter string

const (
	SoundSqueak SoundCharacter = "Squeak"
	SoundMoo    SoundCharacter = "Moo"
	SoundMeow   SoundCharacter = "Meow"
	SoundPop    SoundCharacter = "Pop"
	SoundChime  SoundCharacter = "Chime"
)

// GenerateWavBytes gera o arquivo WAV completo em memória para o som selecionado.
func GenerateWavBytes(char SoundCharacter) []byte {
	var pcm []int16
	sampleRate := 44100

	switch char {
	case SoundSqueak:
		pcm = buildSqueakPcm(sampleRate)
	case SoundMoo:
		pcm = buildMooPcm(sampleRate)
	case SoundMeow:
		pcm = buildMeowPcm(sampleRate)
	case SoundPop:
		pcm = buildPopPcm(sampleRate)
	case SoundChime:
		pcm = buildChimePcm(sampleRate)
	default:
		pcm = buildSqueakPcm(sampleRate)
	}

	return createWav(pcm, sampleRate)
}

func buildSqueakPcm(sampleRate int) []int16 {
	duration := 0.15
	numSamples := int(float64(sampleRate) * duration)
	pcm := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		frequency := 1200.0 + (2000.0 * (t / duration))
		volume := math.Sin(math.Pi * t / duration)
		angle := 2.0 * math.Pi * frequency * t
		pcm[i] = int16(25000.0 * volume * math.Sin(angle))
	}
	return pcm
}

func buildMooPcm(sampleRate int) []int16 {
	duration := 0.65
	numSamples := int(float64(sampleRate) * duration)
	pcm := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		progress := t / duration

		vibrato := math.Sin(2.0*math.Pi*6.0*t) * 4.0
		frequency := (110.0 - (25.0 * progress)) + vibrato

		volume := math.Min(1.0, t/0.05) * math.Min(1.0, (duration-t)/0.2)

		angle := 2.0 * math.Pi * frequency * t
		fundamental := math.Sin(angle)
		secondHarmonic := 0.4 * math.Sin(2.0*angle)
		thirdHarmonic := 0.2 * math.Sin(3.0*angle)

		pcm[i] = int16(18000.0 * volume * (fundamental + secondHarmonic + thirdHarmonic))
	}
	return pcm
}

func buildMeowPcm(sampleRate int) []int16 {
	duration := 0.4
	numSamples := int(float64(sampleRate) * duration)
	pcm := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		progress := t / duration

		frequency := 500.0 + (500.0 * math.Sin(math.Pi*progress))
		volume := math.Sin(math.Pi * progress)
		angle := 2.0 * math.Pi * frequency * t

		pcm[i] = int16(20000.0 * volume * math.Sin(angle))
	}
	return pcm
}

func buildPopPcm(sampleRate int) []int16 {
	duration := 0.12
	numSamples := int(float64(sampleRate) * duration)
	pcm := make([]int16, numSamples)

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		frequency := 900.0 - (400.0 * (t / duration))
		volume := math.Sin(math.Pi * t / duration)
		angle := 2.0 * math.Pi * frequency * t
		pcm[i] = int16(22000.0 * volume * math.Sin(angle))
	}
	return pcm
}

func buildChimePcm(sampleRate int) []int16 {
	duration := 0.45
	numSamples := int(float64(sampleRate) * duration)
	pcm := make([]int16, numSamples)
	baseFreq := 1046.5 // C6

	for i := 0; i < numSamples; i++ {
		t := float64(i) / float64(sampleRate)
		volume := math.Exp(-6.0 * t / duration)

		fundamental := math.Sin(2.0 * math.Pi * baseFreq * t)
		octave := 0.5 * math.Sin(2.0*math.Pi*baseFreq*2.0*t)

		pcm[i] = int16(20000.0 * volume * (fundamental + octave))
	}
	return pcm
}

// GenerateMelodyWavBytes gera o arquivo WAV da melodia retro.
func GenerateMelodyWavBytes() []byte {
	sampleRate := 22050 // Baixa amostragem para efeito retro
	noteDuration := 0.22
	notes := []float64{
		659.25, 587.33, 523.25, 493.88, 440.00, 493.88, 523.25, 659.25,
		587.33, 493.88, 523.25, 440.00, 0, 440.00, 440.00, 0,
	}

	samplesPerNote := int(float64(sampleRate) * noteDuration)
	numSamples := samplesPerNote * len(notes)
	pcm := make([]int16, numSamples)

	for n, freq := range notes {
		startSample := n * samplesPerNote
		for i := 0; i < samplesPerNote; i++ {
			index := startSample + i
			if freq == 0 {
				pcm[index] = 0
				continue
			}

			t := float64(i) / float64(sampleRate)
			angle := 2.0 * math.Pi * freq * t

			// Timbre chiptune: mistura de onda quadrada (70%) e triangular (30%)
			var square float64
			if math.Sin(angle) >= 0 {
				square = 1.0
			} else {
				square = -1.0
			}

			triangle := math.Abs(math.Mod(angle, 2.0*math.Pi)/math.Pi-1.0)*2.0 - 1.0
			mix := (square * 0.7) + (triangle * 0.3)

			// Decaimento linear de volume por nota
			env := 1.0 - (float64(i) / float64(samplesPerNote))

			pcm[index] = int16(5000.0 * mix * env)
		}
	}

	return createWav(pcm, sampleRate)
}

func createWav(pcm []int16, sampleRate int) []byte {
	bytesPerSample := 2
	numChannels := 1
	subchunk2Size := len(pcm) * bytesPerSample
	chunkSize := 36 + subchunk2Size

	buf := new(bytes.Buffer)

	// RIFF descriptor
	buf.Write([]byte("RIFF"))
	binary.Write(buf, binary.LittleEndian, int32(chunkSize))
	buf.Write([]byte("WAVE"))

	// fmt sub-chunk
	buf.Write([]byte("fmt "))
	binary.Write(buf, binary.LittleEndian, int32(16))                  // Subchunk1Size
	binary.Write(buf, binary.LittleEndian, int16(1))                   // AudioFormat (PCM = 1)
	binary.Write(buf, binary.LittleEndian, int16(numChannels))         // NumChannels
	binary.Write(buf, binary.LittleEndian, int32(sampleRate))          // SampleRate
	binary.Write(buf, binary.LittleEndian, int32(sampleRate*2))        // ByteRate (SampleRate * NumChannels * BitsPerSample/8)
	binary.Write(buf, binary.LittleEndian, int16(numChannels*2))       // BlockAlign
	binary.Write(buf, binary.LittleEndian, int16(16))                  // BitsPerSample

	// data sub-chunk
	buf.Write([]byte("data"))
	binary.Write(buf, binary.LittleEndian, int32(subchunk2Size))

	// Escrita dos dados PCM
	for _, sample := range pcm {
		binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}
