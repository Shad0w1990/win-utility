package main

import (
	"os/exec"
	"path/filepath"
)

func playSelectedSound(filePath string) {
	if !uiSoundEnabled {
		return
	}
	go func() {
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			return
		}
		// بستن فایل‌های صوتی قبلی که احتمالاً باز مانده‌اند
		exec.Command("cmd", "/c", "powershell", "-c", "(New-Object Media.SoundPlayer '"+absPath+"').PlaySync()").Run()
	}()
}
