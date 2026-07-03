package app

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AppSettings representa as preferências salvas do usuário.
type AppSettings struct {
	PetID                string  `json:"petId"`
	Size                 float64 `json:"size"`
	SqueakEnabled        bool    `json:"squeakEnabled"`
	MusicEnabled         bool    `json:"musicEnabled"`
	AudioReactiveEnabled bool    `json:"audioReactiveEnabled"`
	Opacity              float64 `json:"opacity"`
	Topmost              bool    `json:"topmost"`
}

// DefaultSettings retorna as configurações padrões.
func DefaultSettings() AppSettings {
	return AppSettings{
		PetID:                "rato",
		Size:                 200,
		SqueakEnabled:        true,
		MusicEnabled:         false,
		AudioReactiveEnabled: false,
		Opacity:              1.0,
		Topmost:              true,
	}
}

// GetSettingsPath retorna o caminho absoluto para o arquivo de configurações.
func GetSettingsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "ratinhodesktop", "settings.json"), nil
}

// LoadSettings carrega as configurações do disco ou retorna os padrões em caso de erro/inexistência.
func LoadSettings() AppSettings {
	settingsPath, err := GetSettingsPath()
	if err != nil {
		return DefaultSettings()
	}

	file, err := os.Open(settingsPath)
	if err != nil {
		return DefaultSettings()
	}
	defer file.Close()

	var settings AppSettings
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&settings); err != nil {
		return DefaultSettings()
	}

	return settings
}

// SaveSettings grava as configurações atuais no disco.
func SaveSettings(settings AppSettings) error {
	settingsPath, err := GetSettingsPath()
	if err != nil {
		return err
	}

	// Garante que o diretório pai existe
	dir := filepath.Dir(settingsPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	file, err := os.Create(settingsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(settings)
}
