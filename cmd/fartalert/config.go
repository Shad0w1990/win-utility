package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type AppSettings struct {
	Language     string `json:"language"`
	SoundEnabled bool   `json:"sound_enabled"`
	PowerMode    string `json:"power_mode"`
	GpuMode      string `json:"gpu_mode"`
	Volume       int    `json:"volume"`
	Brightness   int    `json:"brightness"`
}

var Settings = AppSettings{
	Language:     "EN",
	SoundEnabled: true,
	PowerMode:    "Balanced",
	GpuMode:      "Standard",
	Volume:       50,
	Brightness:   80,
}

func getConfigPath() string {
	appData := os.Getenv("LOCALAPPDATA")
	dir := filepath.Join(appData, "SysGuard")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "settings.json")
}

func loadSettings() {
	data, err := os.ReadFile(getConfigPath())
	if err == nil {
		json.Unmarshal(data, &Settings)

		uiLanguage = Settings.Language
		uiSoundEnabled = Settings.SoundEnabled
		currentPowerMode = Settings.PowerMode
		currentGpuMode = Settings.GpuMode
		sysVolumeVal = Settings.Volume
		sysBrightnessVal = Settings.Brightness
	}
}

func saveSettings() {
	Settings.Language = uiLanguage
	Settings.SoundEnabled = uiSoundEnabled
	Settings.PowerMode = currentPowerMode
	Settings.GpuMode = currentGpuMode
	Settings.Volume = sysVolumeVal
	Settings.Brightness = sysBrightnessVal

	b, _ := json.MarshalIndent(Settings, "", "  ")
	os.WriteFile(getConfigPath(), b, 0644)
}
