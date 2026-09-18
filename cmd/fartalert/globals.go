package main

import "syscall"

var (
	currentPage                                           = 0
	hwndMain, hwndWidget, hwndChkWidget                   syscall.Handle
	hwndTabDash, hwndTabNet, hwndTabPower, hwndTabGamepad syscall.Handle
	hwndTabTools, hwndTabSettings                         syscall.Handle

	// هندل دکمه‌های نگهداری سیستم
	hwndBtnSound, hwndBtnDiag, hwndBtnAnalyze, hwndBtnClean syscall.Handle
	hwndBtnEventLog                                         syscall.Handle

	hwndBtnSysInfo syscall.Handle

	hwndBtnSilent, hwndBtnBalanced, hwndBtnTurbo syscall.Handle
	hwndBtnEco, hwndBtnStandard, hwndBtnUltra    syscall.Handle

	hwndBtnLang, hwndBtnSoundToggle, hwndBtnStartup syscall.Handle

	hwndBtnPwrSave, hwndBtnPwrBal, hwndBtnPwrHigh syscall.Handle

	hFontNormal, hFontTitle, hFontWidget, hFontSmall syscall.Handle // فونت کوچک برای توضیحات دکمه‌ها
	hBrushBg                                         syscall.Handle
	isWidgetActive                                   = false

	isAsus    = false
	isLaptop  = false
	sysUptime = "0h 0m"

	isDraggingBrightness = false
	isDraggingVolume     = false

	uiLanguage     = "EN"
	uiSoundEnabled = true
	runAtStartup   = false

	currentPowerMode = "Balanced"
	currentGpuMode   = "Standard"
	batteryLevel     = "79.0%"
	batteryLimit     = "80%"

	sysModelName                                                   = "Detecting System..."
	sysCpuName                                                     = "Detecting CPU..."
	sysGpuName                                                     = "Detecting GPU..."
	sysGpuPower                                                    = "0 W"
	totalRamGB, diskTotalGB, diskFreeGB                            float64
	cpuUsageVal, ramUsageVal, gpuLoadVal, gpuTempVal, diskUsageVal float64
	diskMediaType                                                  = "SSD"

	cpuHistory  [60]float64
	ramHistory  [60]float64
	gpuHistory  [60]float64
	diskHistory [60]float64

	netDLHistory [60]float64
	netULHistory [60]float64
	maxNetSpeed  float64 = 1.0

	netUpdateText                  = "Windows Update: Scanning..."
	netTopAppsText                 = "Scanning active connections..."
	currentDLText                  = "DL: 0 KB/s"
	currentULText                  = "UL: 0 KB/s"
	prevIdle, prevKernel, prevUser uint64

	globalHue       float64 = 0.0
	loadingButtonID uint32  = 0

	sysBatteryPercent = "Scanning..."
	sysPowerPlan      = "Scanning..."
	sysPowerPlanGUID  = ""

	sysBrightnessVal = 50
	sysVolumeVal     = 50

	sysGamepadName = "Searching..."
	sysIPAddress   = "IPv4: Detecting..."

	telemetryHistory []TelemetryData
)

type TelemetryData struct {
	Timestamp int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	RAM       float64 `json:"ram"`
	GPU       float64 `json:"gpu"`
	Disk      float64 `json:"disk"`
}

func T(en, fa string) string {
	if uiLanguage == "FA" {
		return fa
	}
	return en
}
