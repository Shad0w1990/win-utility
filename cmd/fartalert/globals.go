package main

import "syscall"

var (
	currentPage                                             = 0
	hwndMain, hwndWidget, hwndChkWidget                     syscall.Handle
	hwndTabDash, hwndTabNet, hwndTabPower, hwndTabGamepad   syscall.Handle
	hwndTabTools, hwndTabSettings                           syscall.Handle
	hwndBtnSound, hwndBtnDiag, hwndBtnAnalyze, hwndBtnClean syscall.Handle
	hwndBtnSysInfo                                          syscall.Handle

	hwndBtnSilent, hwndBtnBalanced, hwndBtnTurbo syscall.Handle
	hwndBtnEco, hwndBtnStandard, hwndBtnUltra    syscall.Handle

	// هندل دکمه‌های صفحه تنظیمات
	hwndBtnLang, hwndBtnSoundToggle, hwndBtnStartup syscall.Handle

	hFontNormal, hFontTitle, hFontWidget syscall.Handle
	hBrushBg                             syscall.Handle
	isWidgetActive                       = false
	isAsusLaptop                         = false

	// متغیرهای بخش تنظیمات
	uiLanguage     = "EN"  // زبان پیش‌فرض
	uiSoundEnabled = true  // صدای کلیک‌ها
	runAtStartup   = false // وضعیت اجرای خودکار

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
	sysBrightness     = "Scanning..."
	sysVolume         = "OS Managed"

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

// تابع جادویی ترجمه: اگر زبان فارسی باشد متن دوم را برمی‌گرداند
func T(en, fa string) string {
	if uiLanguage == "FA" {
		return fa
	}
	return en
}
