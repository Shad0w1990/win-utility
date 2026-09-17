package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

// مدیریت رجیستری برای اجرای خودکار
func checkStartup() {
	out, _ := exec.Command("cmd", "/c", `reg query HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v SysGuard`).Output()
	if strings.Contains(string(out), "SysGuard") {
		runAtStartup = true
	}
}

func toggleStartup() {
	exe, _ := os.Executable()
	if runAtStartup {
		exec.Command("cmd", "/c", `reg delete HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v SysGuard /f`).Run()
		runAtStartup = false
	} else {
		exec.Command("cmd", "/c", fmt.Sprintf(`reg add HKCU\Software\Microsoft\Windows\CurrentVersion\Run /v SysGuard /t REG_SZ /d "\"%s\"" /f`, exe)).Run()
		runAtStartup = true
	}
}

// تابع صدای تمام دکمه‌ها
func playUIClick() {
	if uiSoundEnabled {
		go procBeep.Call(uintptr(700), uintptr(30)) // یک تیک نرم و کوتاه
	}
}

func updateTabButtonsVisibility() {
	showG := (isAsusLaptop && currentPage == 2)
	swG := SW_HIDE
	if showG {
		swG = SW_SHOW
	}

	procShowWindow.Call(uintptr(hwndBtnSilent), uintptr(swG))
	procShowWindow.Call(uintptr(hwndBtnBalanced), uintptr(swG))
	procShowWindow.Call(uintptr(hwndBtnTurbo), uintptr(swG))
	procShowWindow.Call(uintptr(hwndBtnEco), uintptr(swG))
	procShowWindow.Call(uintptr(hwndBtnStandard), uintptr(swG))
	procShowWindow.Call(uintptr(hwndBtnUltra), uintptr(swG))

	swD := SW_HIDE
	if currentPage == 0 {
		swD = SW_SHOW
	}
	procShowWindow.Call(uintptr(hwndBtnSysInfo), uintptr(swD))

	// دکمه‌های تب تنظیمات
	showS := (isAsusLaptop && currentPage == 5) || (!isAsusLaptop && currentPage == 4)
	swS := SW_HIDE
	if showS {
		swS = SW_SHOW
	}
	procShowWindow.Call(uintptr(hwndBtnLang), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnSoundToggle), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnStartup), uintptr(swS))
}

func wndProc(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_ERASEBKGND:
		return 1
	case WM_CREATE:
		brushPtr, _, _ := procCreateSolidBrush.Call(uintptr(RGB(10, 14, 25)))
		hBrushBg = syscall.Handle(brushPtr)

		checkStartup() // بررسی استارت‌آپ هنگام لود برنامه

		procSetTimer.Call(uintptr(hwnd), 1, 30, 0)
		startHardwareScanner()
		startNetworkScanner()
		startGraphScanner()
		startToolsScanner()

		InitSysInfoWindow()

		createWidget()
		updateTabButtonsVisibility()

	case WM_TIMER:
		globalHue += 3.5
		if globalHue >= 360.0 {
			globalHue -= 360.0
		}

		if isWidgetActive {
			procInvalidateRect.Call(uintptr(hwndWidget), 0, 1)
		}

		if (isAsusLaptop && currentPage == 3) || (!isAsusLaptop && currentPage == 2) {
			procInvalidateRect.Call(uintptr(hwnd), 0, 1)
		}

		activeBtn := atomic.LoadUint32(&loadingButtonID)
		if activeBtn != 0 {
			var hBtn syscall.Handle
			switch activeBtn {
			case IDC_BTN_SOUND:
				hBtn = hwndBtnSound
			case IDC_BTN_DIAG:
				hBtn = hwndBtnDiag
			case IDC_BTN_ANALYZE:
				hBtn = hwndBtnAnalyze
			case IDC_BTN_CLEAN:
				hBtn = hwndBtnClean
			case IDC_BTN_SYSINFO:
				hBtn = hwndBtnSysInfo
			case IDC_MODE_SILENT:
				hBtn = hwndBtnSilent
			case IDC_MODE_BALANCED:
				hBtn = hwndBtnBalanced
			case IDC_MODE_TURBO:
				hBtn = hwndBtnTurbo
			case IDC_GPU_ECO:
				hBtn = hwndBtnEco
			case IDC_GPU_STD:
				hBtn = hwndBtnStandard
			case IDC_GPU_ULTRA:
				hBtn = hwndBtnUltra
			}
			if hBtn != 0 {
				procInvalidateRect.Call(uintptr(hBtn), 0, 1)
			}
		}

	case WM_CTLCOLORSTATIC:
		hdc := syscall.Handle(wParam)
		procSetBkMode.Call(uintptr(hdc), uintptr(TRANSPARENT))
		procSetTextColor.Call(uintptr(hdc), uintptr(RGB(180, 200, 220)))
		return uintptr(hBrushBg)

	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis.CtlType == ODT_BUTTON {
			if dis.CtlID == IDC_BTN_SYSINFO {
				currentLoading := atomic.LoadUint32(&loadingButtonID)
				isDown := (dis.ItemState & ODS_SELECTED) != 0
				borderColor := RGB(0, 255, 150)
				fillColor := RGB(15, 20, 30)
				txtColor := borderColor
				text := T("📄 Full System Info", "📄 اطلاعات جامع قطعات")

				if currentLoading == dis.CtlID {
					rgbColor := HSVtoRGB(globalHue, 1.0, 1.0)
					borderColor, txtColor = rgbColor, rgbColor
					text = T("⏳ Loading...", "⏳ در حال ساخت گزارش...")
				} else if isDown {
					fillColor = borderColor
					txtColor = RGB(10, 14, 25)
				}

				drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 14, borderColor)
				if currentLoading != dis.CtlID && fillColor != borderColor {
					drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 12, fillColor)
				}
				procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
				procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
				procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
				DrawTextSafe(uintptr(dis.Hdc), text, &dis.RcItem, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
			} else {
				drawCustomButton(dis)
			}
			return 1
		}

	case WM_COMMAND:
		if int(wParam>>16) == BN_CLICKED {
			playUIClick() // پخش صدای کلیک در صورت فعال بودن تنظیمات
			cmdID := int(wParam & 0xFFFF)
			switch cmdID {
			case IDC_TAB_DASH:
				currentPage = 0
				procShowWindow.Call(uintptr(hwndChkWidget), SW_SHOW)
				updateTabButtonsVisibility()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_TAB_NET:
				currentPage = 1
				procShowWindow.Call(uintptr(hwndChkWidget), SW_HIDE)
				updateTabButtonsVisibility()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_TAB_POWER:
				if isAsusLaptop {
					currentPage = 2
					procShowWindow.Call(uintptr(hwndChkWidget), SW_HIDE)
					updateTabButtonsVisibility()
					procInvalidateRect.Call(uintptr(hwnd), 0, 1)
				}
			case IDC_TAB_GAMEPAD:
				if isAsusLaptop {
					currentPage = 3
				} else {
					currentPage = 2
				}
				procShowWindow.Call(uintptr(hwndChkWidget), SW_HIDE)
				updateTabButtonsVisibility()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_TAB_TOOLS:
				if isAsusLaptop {
					currentPage = 4
				} else {
					currentPage = 3
				}
				procShowWindow.Call(uintptr(hwndChkWidget), SW_HIDE)
				updateTabButtonsVisibility()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_TAB_SETTINGS: // کلیک روی تب تنظیمات
				if isAsusLaptop {
					currentPage = 5
				} else {
					currentPage = 4
				}
				procShowWindow.Call(uintptr(hwndChkWidget), SW_HIDE)
				updateTabButtonsVisibility()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)

			case IDC_CHK_WIDGET:
				state, _, _ := procSendMessage.Call(uintptr(hwndChkWidget), BM_GETCHECK, 0, 0)
				if state == BST_CHECKED {
					procShowWindow.Call(uintptr(hwndWidget), SW_SHOW)
					isWidgetActive = true
				} else {
					procShowWindow.Call(uintptr(hwndWidget), SW_HIDE)
					isWidgetActive = false
				}

			case IDC_SET_LANG:
				if uiLanguage == "EN" {
					uiLanguage = "FA"
				} else {
					uiLanguage = "EN"
				}
				procInvalidateRect.Call(uintptr(hwnd), 0, 1) // رفرش کل صفحه برای تغییر زبان
			case IDC_SET_SOUND:
				uiSoundEnabled = !uiSoundEnabled
				procInvalidateRect.Call(uintptr(hwndBtnSoundToggle), 0, 1)
			case IDC_SET_STARTUP:
				toggleStartup()
				procInvalidateRect.Call(uintptr(hwndBtnStartup), 0, 1)

			case IDC_MODE_SILENT:
				currentPowerMode = "Silent"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_MODE_BALANCED:
				currentPowerMode = "Balanced"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_MODE_TURBO:
				currentPowerMode = "Turbo"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_ECO:
				currentGpuMode = "Eco"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_STD:
				currentGpuMode = "Standard"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_ULTRA:
				currentGpuMode = "Ultimate"
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)

			case IDC_BTN_SYSINFO:
				OpenSysInfoWindow()

			case IDC_BTN_SOUND:
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_BTN_SOUND)
					go func() {
						go procBeep.Call(uintptr(400), uintptr(250))
						time.Sleep(800 * time.Millisecond)
						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndBtnSound), 0, 1)
					}()
				}
			case IDC_BTN_DIAG:
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_BTN_DIAG)
					go func() {
						cmdTime := exec.Command("cmd", "/c", "w32tm /resync")
						cmdTime.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						cmdTime.Run()

						cmdDNS := exec.Command("cmd", "/c", "ipconfig /flushdns")
						cmdDNS.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						cmdDNS.Run()

						time.Sleep(1 * time.Second)

						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndBtnDiag), 0, 1)

						ShowMessageBox(hwndMain, T("System Time synced and DNS Cache flushed successfully.", "ساعت ویندوز و کش اینترنت با موفقیت همگام‌سازی شدند."), T("Force Sync", "همگام‌سازی"), 0x00000040)
					}()
				}
			case IDC_BTN_ANALYZE:
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_BTN_ANALYZE)
					go func() {
						time.Sleep(1500 * time.Millisecond)

						var cAvg, rAvg, gAvg, dAvg float64
						count := float64(len(telemetryHistory))

						if count > 0 {
							for _, t := range telemetryHistory {
								cAvg += t.CPU
								rAvg += t.RAM
								gAvg += t.GPU
								dAvg += t.Disk
							}
							cAvg /= count
							rAvg /= count
							gAvg /= count
							dAvg /= count
						} else {
							cAvg = cpuUsageVal
							rAvg = ramUsageVal
							gAvg = gpuLoadVal
							dAvg = diskUsageVal
						}

						type Component struct {
							Name   string
							Score  float64
							Reason string
						}
						comps := []Component{}

						cScore := cAvg
						cReason := fmt.Sprintf(T("Average Load: %.1f%% - ", "فشار پردازشی: %.1f%% - "), cAvg)
						if cAvg > 85 {
							cScore += 30
							cReason += T("CRITICAL: Processor is heavily bottlenecking the system.", "بحرانی: پردازنده به شدت ضعیف است و باعث افت فریم می‌شود.")
						} else if cAvg > 60 {
							cReason += T("Moderate load. Good for now.", "فشار متوسط است. در حال حاضر مشکلی ندارد.")
						} else {
							cReason += T("Excellent. Plenty of processing headroom.", "عالی. پردازنده قدرت بسیار زیادی برای کارهای سنگین دارد.")
						}
						comps = append(comps, Component{T("Processor (CPU)", "پردازنده اصلی (CPU)"), cScore, cReason})

						rScore := rAvg
						rReason := fmt.Sprintf(T("Average Usage: %.1f%% - ", "میانگین مصرف: %.1f%% - "), rAvg)
						if rAvg > 85 {
							rScore += 40
							rReason += T("CRITICAL: Running out of memory! Upgrade RAM immediately.", "بحرانی: ظرفیت رم در حال پر شدن است! حتماً رم را ارتقا دهید.")
						} else if totalRamGB <= 8 && rAvg > 60 {
							rScore += 20
							rReason += T("8GB is too low for modern apps. Upgrade to 16GB+ recommended.", "۸ گیگابایت رم برای برنامه‌های امروزی کم است. ارتقا پیشنهاد می‌شود.")
						} else {
							rReason += T("Memory capacity is sufficient.", "ظرفیت رم کاملاً جوابگوی نیازهای شماست.")
						}
						comps = append(comps, Component{T("Memory (RAM)", "حافظه موقت (RAM)"), rScore, rReason})

						gScore := gAvg
						gReason := fmt.Sprintf(T("Average Load: %.1f%% - ", "فشار پردازشی: %.1f%% - "), gAvg)
						if gAvg > 90 {
							gScore += 25
							gReason += T("High graphics load. Upgrade if experiencing gaming lag.", "فشار گرافیکی بالا. اگر در بازی‌ها لگ دارید گرافیک را ارتقا دهید.")
						} else {
							gReason += T("Graphics performance is optimal.", "عملکرد کارت گرافیک در وضعیت ایده‌آل است.")
						}
						comps = append(comps, Component{T("Graphics (GPU)", "کارت گرافیک (GPU)"), gScore, gReason})

						dScore := dAvg
						dReason := fmt.Sprintf(T("Space Used: %.1f%% - ", "فضای پر شده: %.1f%% - "), dAvg)
						if dAvg > 90 {
							dScore += 45
							dReason += T("CRITICAL: Drive is almost full! Causes severe OS lag.", "بحرانی: درایو ویندوز پر شده است که باعث هنگی شدید سیستم می‌شود.")
						} else if strings.Contains(strings.ToUpper(diskMediaType), "HDD") {
							dScore += 60
							dReason += T("CRITICAL: OS is on an HDD. Upgrade to SSD for massive speed boost.", "بحرانی: ویندوز روی هارد دیسک معمولی نصب است. خرید SSD سرعت را چند برابر می‌کند.")
						} else {
							dReason += T("Storage health and capacity are good.", "وضعیت حافظه و درایوها بسیار عالی است.")
						}
						comps = append(comps, Component{T("Storage (Disk)", "حافظه ذخیره‌سازی (SSD/HDD)"), dScore, dReason})

						sort.Slice(comps, func(i, j int) bool { return comps[i].Score > comps[j].Score })

						report := T("🧠 AI Hardware Diagnostic Report\n", "🧠 گزارش تحلیل هوشمند سخت‌افزار\n")
						if count > 0 {
							report += fmt.Sprintf(T("Based on %d historical data points (last 30 days).\n\n", "بر اساس %d نقطه داده در 30 روز گذشته.\n\n"), int(count))
						} else {
							report += T("Based on current real-time snapshot.\n\n", "بر اساس اسکن زنده (برای دقت بالاتر برنامه را باز بگذارید).\n\n")
						}

						report += T("⚠️ UPGRADE PRIORITY RANKING (1 = Most Urgent):\n\n", "⚠️ رتبه‌بندی قطعات ضعیف جهت ارتقاء:\n\n")
						for i, c := range comps {
							report += fmt.Sprintf(T("Rank %d: %s\n   └ %s\n\n", "رتبه %d: %s\n   └ %s\n\n"), i+1, c.Name, c.Reason)
						}

						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndBtnAnalyze), 0, 1)

						ShowMessageBox(hwndMain, report, T("Smart Hardware Analysis", "تحلیل هوش مصنوعی"), 0x00000040)
					}()
				}
			case IDC_BTN_CLEAN:
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_BTN_CLEAN)
					go func() {
						time.Sleep(1 * time.Second)
						tempPath := os.TempDir()
						entries, _ := os.ReadDir(tempPath)
						deleted := 0
						for _, entry := range entries {
							if err := os.RemoveAll(filepath.Join(tempPath, entry.Name())); err == nil {
								deleted++
							}
						}
						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndBtnClean), 0, 1)
						ShowMessageBox(hwndMain, fmt.Sprintf(T("Cleaned %d junk files/folders.", "%d فایل موقت و بی‌مصرف با موفقیت پاک شد."), deleted), T("Optimizer", "بهینه‌ساز سیستم"), 0x00000040)
					}()
				}
			}
		}

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			var rect RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

			drawRect(hdc, 0, 0, int(rect.Right), int(rect.Bottom), 0, RGB(10, 14, 25))
			drawRect(hdc, 0, 0, 180, int(rect.Bottom), 0, RGB(5, 7, 12))

			contentRect := RECT{Left: 220, Top: 40, Right: rect.Right - 20, Bottom: rect.Bottom}
			procSelectObject.Call(hdc, uintptr(hFontTitle))
			setTextColor(hdc, RGB(0, 210, 255))

			if currentPage == 0 {
				DrawTextSafe(hdc, T("SYSTEM DASHBOARD  |  ", "داشبورد سخت افزار  |  ")+sysModelName, &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)
				contentW := int(rect.Right) - 260
				graphW := 280
				graphH := 65

				drawRect(hdc, 210, 85, int(rect.Right)-30, 185, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				contentRect.Top, contentRect.Left = 93, 225
				DrawTextSafe(hdc, fmt.Sprintf("CPU: %s", sysCpuName), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				contentRect.Top = 120
				DrawTextSafe(hdc, fmt.Sprintf(T("Utilization: %.1f%%", "مصرف پردازنده: %.1f%%"), cpuUsageVal), &contentRect, DT_LEFT|DT_TOP)
				drawGridGraph(hdc, 210+contentW-graphW+10, 98, graphW, graphH, cpuHistory[:], 100, RGB(0, 210, 255), RGB(10, 50, 70))

				drawRect(hdc, 210, 195, int(rect.Right)-30, 295, 12, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				contentRect.Top = 203
				DrawTextSafe(hdc, fmt.Sprintf(T("Memory: %.1f GB Total (DDR5)", "حافظه رم: %.1f گیگابایت"), totalRamGB), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				contentRect.Top = 230
				DrawTextSafe(hdc, fmt.Sprintf(T("In Use: %.1f%%", "مقدار مصرفی: %.1f%%"), ramUsageVal), &contentRect, DT_LEFT|DT_TOP)
				drawGridGraph(hdc, 210+contentW-graphW+10, 208, graphW, graphH, ramHistory[:], 100, RGB(0, 255, 150), RGB(10, 60, 30))

				drawRect(hdc, 210, 305, int(rect.Right)-30, 405, 12, panelColor)
				setTextColor(hdc, RGB(255, 140, 50))
				contentRect.Top = 313
				DrawTextSafe(hdc, fmt.Sprintf("GPU: %s", sysGpuName), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				contentRect.Top = 340
				DrawTextSafe(hdc, fmt.Sprintf(T("Usage: %.0f%%  |  Power: %s", "مصرف گرافیک: %.0f%%  |  برق: %s"), gpuLoadVal, sysGpuPower), &contentRect, DT_LEFT|DT_TOP)
				drawGridGraph(hdc, 210+contentW-graphW-110, 318, graphW-40, graphH, gpuHistory[:], 100, RGB(255, 140, 50), RGB(60, 25, 15))
				drawTempGauge(hdc, int(rect.Right)-95, 355, 45, gpuTempVal)

				drawRect(hdc, 210, 415, int(rect.Right)-30, 515, 12, panelColor)
				setTextColor(hdc, RGB(200, 80, 255))
				contentRect.Top = 423
				DrawTextSafe(hdc, fmt.Sprintf(T("Drive C: (%.0f GB Total [%s])", "درایو ویندوز: (حجم کل %.0f گیگابایت [%s])"), diskTotalGB, diskMediaType), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				contentRect.Top = 450
				DrawTextSafe(hdc, fmt.Sprintf(T("Free: %.1f GB (%.0f%% Used)", "آزاد: %.1f گیگابایت (%.0f%% پر شده)"), diskFreeGB, diskUsageVal), &contentRect, DT_LEFT|DT_TOP)
				drawGridGraph(hdc, 210+contentW-graphW+10, 428, graphW, graphH, diskHistory[:], 100, RGB(200, 80, 255), RGB(50, 15, 60))

			} else if currentPage == 1 {
				DrawTextSafe(hdc, T("ADVANCED NETWORK TRAFFIC", "ترافیک لحظه‌ای شبکه و اینترنت"), &contentRect, DT_LEFT|DT_TOP)

				procSelectObject.Call(hdc, uintptr(hFontWidget))
				setTextColor(hdc, RGB(0, 255, 150))
				rIP := RECT{Left: int32(rect.Right - 220), Top: 48, Right: rect.Right - 30, Bottom: 70}
				DrawTextSafe(hdc, sysIPAddress, &rIP, DT_RIGHT|DT_TOP)

				procSelectObject.Call(hdc, uintptr(hFontNormal))

				drawRect(hdc, 210, 90, int(rect.Right)-30, 145, 10, RGB(16, 22, 35))
				drawRect(hdc, 210, 155, int(rect.Right)-30, 245, 10, RGB(16, 22, 35))

				setTextColor(hdc, RGB(255, 120, 120))
				contentRect.Top = 110
				contentRect.Left = 230
				DrawTextSafe(hdc, netUpdateText, &contentRect, DT_LEFT|DT_TOP)

				setTextColor(hdc, RGB(0, 210, 255))
				contentRect.Top = 170
				DrawTextSafe(hdc, T("Top active connections (ESTABLISHED):", "نرم‌افزارهایی که بیشترین اتصال به اینترنت را دارند:"), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(180, 200, 220))
				contentRect.Top = 195
				DrawTextSafe(hdc, netTopAppsText, &contentRect, DT_LEFT|DT_TOP)

				setTextColor(hdc, RGB(0, 210, 255))
				contentRect.Top = 265
				DrawTextSafe(hdc, "▼ "+currentDLText, &contentRect, DT_LEFT|DT_TOP)

				setTextColor(hdc, RGB(255, 80, 200))
				contentRect.Left = 400
				DrawTextSafe(hdc, "▲ "+currentULText, &contentRect, DT_LEFT|DT_TOP)

				drawAdvancedNetworkGraph(hdc, 210, 290, int(rect.Right)-240, 180, netDLHistory[:], netULHistory[:], maxNetSpeed)

			} else if isAsusLaptop && currentPage == 2 {
				DrawTextSafe(hdc, "G-HELPER CONTROL PANEL", &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)

				drawRect(hdc, 210, 95, int(rect.Right)-30, 210, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				contentRect.Top, contentRect.Left = 110, 230
				DrawTextSafe(hdc, fmt.Sprintf("⚡ Operating Mode: %s", currentPowerMode), &contentRect, DT_LEFT|DT_TOP)

				drawRect(hdc, 210, 225, int(rect.Right)-30, 340, 12, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				contentRect.Top = 240
				DrawTextSafe(hdc, fmt.Sprintf("🎮 GPU Ultimate Control: %s", currentGpuMode), &contentRect, DT_LEFT|DT_TOP)

				drawRect(hdc, 210, 355, int(rect.Right)-30, 465, 12, panelColor)
				setTextColor(hdc, RGB(255, 180, 0))
				contentRect.Top = 370
				DrawTextSafe(hdc, fmt.Sprintf("🔋 Battery Charge Limit: %s (Current: %s)", batteryLimit, batteryLevel), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(180, 200, 220))
				contentRect.Top = 405
				DrawTextSafe(hdc, "Protecting battery lifespan with automated charge capping.", &contentRect, DT_LEFT|DT_TOP)

			} else if (isAsusLaptop && currentPage == 3) || (!isAsusLaptop && currentPage == 2) {

				DrawTextSafe(hdc, T("GAMEPAD TESTER (PRO MODE)", "تست و مانیتورینگ دقیق دسته بازی"), &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				var state XINPUT_STATE
				ret, _, _ := procXInputGetState.Call(0, uintptr(unsafe.Pointer(&state)))
				cGray := RGB(45, 55, 75)
				cGreen := RGB(0, 255, 150)
				cRed := RGB(255, 80, 100)
				cBlue := RGB(0, 210, 255)
				cDark := RGB(10, 14, 20)

				centerX, centerY := 560, 310

				if ret == 0 {
					btns := state.Gamepad.wButtons

					drawRect(hdc, centerX-200, centerY-60, centerX+200, centerY+60, 60, RGB(20, 26, 40))
					drawRect(hdc, centerX-230, centerY-10, centerX-110, centerY+140, 60, RGB(20, 26, 40))
					drawRect(hdc, centerX+110, centerY-10, centerX+230, centerY+140, 60, RGB(20, 26, 40))
					drawRect(hdc, centerX-110, centerY-50, centerX+110, centerY+70, 20, RGB(12, 16, 25))

					ltVal := state.Gamepad.bLeftTrigger
					rtVal := state.Gamepad.bRightTrigger
					ltFill := int((float64(ltVal) / 255.0) * 100)
					rtFill := int((float64(rtVal) / 255.0) * 100)

					drawRect(hdc, centerX-200, centerY-130, centerX-100, centerY-110, 10, cDark)
					if ltFill > 0 {
						drawRect(hdc, centerX-200, centerY-130, centerX-200+ltFill, centerY-110, 10, cBlue)
					}
					cRectLT := RECT{Left: int32(centerX - 200), Top: int32(centerY - 155), Right: int32(centerX - 100), Bottom: int32(centerY - 135)}
					DrawTextSafe(hdc, fmt.Sprintf("LT %s: %d", T("Pressure", "فشار"), ltVal), &cRectLT, DT_CENTER|DT_TOP)

					drawRect(hdc, centerX+100, centerY-130, centerX+200, centerY-110, 10, cDark)
					if rtFill > 0 {
						drawRect(hdc, centerX+100, centerY-130, centerX+100+rtFill, centerY-110, 10, cBlue)
					}
					cRectRT := RECT{Left: int32(centerX + 100), Top: int32(centerY - 155), Right: int32(centerX + 200), Bottom: int32(centerY - 135)}
					DrawTextSafe(hdc, fmt.Sprintf("RT %s: %d", T("Pressure", "فشار"), rtVal), &cRectRT, DT_CENTER|DT_TOP)

					lbCol, rbCol := cGray, cGray
					if btns&XINPUT_GAMEPAD_LEFT_SHOULDER != 0 {
						lbCol = cBlue
					}
					if btns&XINPUT_GAMEPAD_RIGHT_SHOULDER != 0 {
						rbCol = cBlue
					}
					drawRect(hdc, centerX-190, centerY-90, centerX-110, centerY-70, 10, lbCol)
					drawRect(hdc, centerX+110, centerY-90, centerX+190, centerY-70, 10, rbCol)

					backCol, startCol := cGray, cGray
					if btns&XINPUT_GAMEPAD_BACK != 0 {
						backCol = cGreen
					}
					if btns&XINPUT_GAMEPAD_START != 0 {
						startCol = cGreen
					}
					drawRect(hdc, centerX-40, centerY-15, centerX-15, centerY, 5, backCol)
					drawRect(hdc, centerX+15, centerY-15, centerX+40, centerY, 5, startCol)

					dpX, dpY := centerX-60, centerY+70
					up, down, left, right := cGray, cGray, cGray, cGray
					if btns&XINPUT_GAMEPAD_DPAD_UP != 0 {
						up = cGreen
					}
					if btns&XINPUT_GAMEPAD_DPAD_DOWN != 0 {
						down = cGreen
					}
					if btns&XINPUT_GAMEPAD_DPAD_LEFT != 0 {
						left = cGreen
					}
					if btns&XINPUT_GAMEPAD_DPAD_RIGHT != 0 {
						right = cGreen
					}

					drawRect(hdc, dpX-12, dpY-35, dpX+12, dpY+35, 3, cDark)
					drawRect(hdc, dpX-35, dpY-12, dpX+35, dpY+12, 3, cDark)
					drawRect(hdc, dpX-10, dpY-32, dpX+10, dpY-12, 3, up)
					drawRect(hdc, dpX-10, dpY+12, dpX+10, dpY+32, 3, down)
					drawRect(hdc, dpX-32, dpY-10, dpX-12, dpY+10, 3, left)
					drawRect(hdc, dpX+12, dpY-10, dpX+32, dpY+10, 3, right)

					fbX, fbY := centerX+140, centerY-10

					lblA, lblB, lblX, lblY := "A", "B", "X", "Y"
					nameLower := strings.ToLower(sysGamepadName)
					if strings.Contains(nameLower, "dual") || strings.Contains(nameLower, "playstation") || strings.Contains(nameLower, "ps4") || strings.Contains(nameLower, "ps5") {
						lblA, lblB, lblX, lblY = "✕", "◯", "■", "▲"
					} else if strings.Contains(nameLower, "nintendo") || strings.Contains(nameLower, "pro controller") || strings.Contains(nameLower, "joy-con") {
						lblA, lblB, lblX, lblY = "B", "A", "Y", "X"
					}

					drawBtnWithText := func(cx, cy int, isPressed bool, baseColor uint32, text string) {
						bgCol := cGray
						txtCol := baseColor
						if isPressed {
							bgCol = baseColor
							txtCol = cDark
						}
						drawCircle(hdc, cx, cy, 14, bgCol)
						procSelectObject.Call(hdc, uintptr(hFontWidget))
						setTextColor(hdc, txtCol)
						rBtn := RECT{Left: int32(cx - 14), Top: int32(cy - 14), Right: int32(cx + 14), Bottom: int32(cy + 14)}
						DrawTextSafe(hdc, text, &rBtn, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
						procSelectObject.Call(hdc, uintptr(hFontNormal))
					}

					drawBtnWithText(fbX, fbY+28, btns&XINPUT_GAMEPAD_A != 0, cGreen, lblA)
					drawBtnWithText(fbX+28, fbY, btns&XINPUT_GAMEPAD_B != 0, cRed, lblB)
					drawBtnWithText(fbX-28, fbY, btns&XINPUT_GAMEPAD_X != 0, cBlue, lblX)
					drawBtnWithText(fbX, fbY-28, btns&XINPUT_GAMEPAD_Y != 0, RGB(255, 215, 0), lblY)

					lsX, lsY := centerX-140, centerY-10
					lsValX := int(state.Gamepad.sThumbLX)
					lsValY := int(state.Gamepad.sThumbLY)
					lsCol := RGB(60, 70, 90)
					if btns&XINPUT_GAMEPAD_LEFT_THUMB != 0 {
						lsCol = cGreen
					}

					drawCircle(hdc, lsX, lsY, 35, cDark)
					drawRect(hdc, lsX-30, lsY-1, lsX+30, lsY+1, 0, RGB(40, 50, 70))
					drawRect(hdc, lsX-1, lsY-30, lsX+1, lsY+30, 0, RGB(40, 50, 70))

					lsOffsetX := int((float64(lsValX) / 32768.0) * 20)
					lsOffsetY := int(-(float64(lsValY) / 32768.0) * 20)
					drawCircle(hdc, lsX+lsOffsetX, lsY+lsOffsetY, 15, lsCol)

					cRectLS := RECT{Left: int32(lsX - 60), Top: int32(lsY + 45), Right: int32(lsX + 60), Bottom: int32(lsY + 85)}
					DrawTextSafe(hdc, fmt.Sprintf("X: %d", lsValX), &cRectLS, DT_CENTER|DT_TOP)
					cRectLS.Top += 15
					DrawTextSafe(hdc, fmt.Sprintf("Y: %d", lsValY), &cRectLS, DT_CENTER|DT_TOP)

					rsX, rsY := centerX+60, centerY+70
					rsValX := int(state.Gamepad.sThumbRX)
					rsValY := int(state.Gamepad.sThumbRY)
					rsCol := RGB(60, 70, 90)
					if btns&XINPUT_GAMEPAD_RIGHT_THUMB != 0 {
						rsCol = cGreen
					}

					drawCircle(hdc, rsX, rsY, 35, cDark)
					drawRect(hdc, rsX-30, rsY-1, rsX+30, rsY+1, 0, RGB(40, 50, 70))
					drawRect(hdc, rsX-1, rsY-30, rsX+1, rsY+30, 0, RGB(40, 50, 70))

					rsOffsetX := int((float64(rsValX) / 32768.0) * 20)
					rsOffsetY := int(-(float64(rsValY) / 32768.0) * 20)
					drawCircle(hdc, rsX+rsOffsetX, rsY+rsOffsetY, 15, rsCol)

					cRectRS := RECT{Left: int32(rsX - 60), Top: int32(rsY + 45), Right: int32(rsX + 60), Bottom: int32(rsY + 85)}
					DrawTextSafe(hdc, fmt.Sprintf("X: %d", rsValX), &cRectRS, DT_CENTER|DT_TOP)
					cRectRS.Top += 15
					DrawTextSafe(hdc, fmt.Sprintf("Y: %d", rsValY), &cRectRS, DT_CENTER|DT_TOP)

					setTextColor(hdc, cGreen)
					contentRect.Top = 110
					DrawTextSafe(hdc, T("🟢 Connected: ", "🟢 متصل شد: ")+sysGamepadName, &contentRect, DT_LEFT|DT_TOP)
				} else {
					setTextColor(hdc, cRed)
					contentRect.Top = 150
					DrawTextSafe(hdc, T("🔴 Waiting for Gamepad...", "🔴 لطفاً دسته را متصل کنید..."), &contentRect, DT_LEFT|DT_TOP)
				}
			} else if (isAsusLaptop && currentPage == 4) || (!isAsusLaptop && currentPage == 3) {
				DrawTextSafe(hdc, T("SYSTEM TOOLS & STATUS", "ابزارهای سیستمی و وضعیت‌ها"), &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)

				drawRect(hdc, 210, 90, 490, 200, 12, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				cRect1 := RECT{Left: 230, Top: 110, Right: 470, Bottom: 190}
				DrawTextSafe(hdc, T("🔋 Battery Status", "🔋 وضعیت باتری"), &cRect1, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				cRect1.Top += 35
				DrawTextSafe(hdc, sysBatteryPercent, &cRect1, DT_LEFT|DT_TOP)

				drawRect(hdc, 510, 90, 790, 200, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				cRect2 := RECT{Left: 530, Top: 110, Right: 770, Bottom: 190}
				DrawTextSafe(hdc, T("⚡ Power Plan", "⚡ مصرف انرژی"), &cRect2, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				cRect2.Top += 35
				DrawTextSafe(hdc, sysPowerPlan, &cRect2, DT_LEFT|DT_TOP)

				drawRect(hdc, 210, 220, 490, 330, 12, panelColor)
				setTextColor(hdc, RGB(255, 180, 0))
				cRect3 := RECT{Left: 230, Top: 240, Right: 470, Bottom: 320}
				DrawTextSafe(hdc, T("☀️ Display Brightness", "☀️ روشنایی مانیتور"), &cRect3, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				cRect3.Top += 35
				DrawTextSafe(hdc, sysBrightness, &cRect3, DT_LEFT|DT_TOP)

				drawRect(hdc, 510, 220, 790, 330, 12, panelColor)
				setTextColor(hdc, RGB(200, 80, 255))
				cRect4 := RECT{Left: 530, Top: 240, Right: 770, Bottom: 320}
				DrawTextSafe(hdc, T("🔊 System Volume", "🔊 وضعیت صدا"), &cRect4, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				cRect4.Top += 35
				DrawTextSafe(hdc, sysVolume, &cRect4, DT_LEFT|DT_TOP)
			} else if (isAsusLaptop && currentPage == 5) || (!isAsusLaptop && currentPage == 4) {
				// --- رندر کردن گرافیک تب جدید تنظیمات ---
				DrawTextSafe(hdc, T("APPLICATION SETTINGS", "تنظیمات نرم افزار"), &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)
				drawRect(hdc, 210, 80, int(rect.Right)-30, 310, 12, panelColor)

				setTextColor(hdc, RGB(220, 235, 255))
				cRectS := RECT{Left: 500, Top: 115, Right: rect.Right - 40, Bottom: 150}
				DrawTextSafe(hdc, T("Switch the application interface language.", "تغییر زبان رابط کاربری نرم افزار."), &cRectS, DT_LEFT|DT_TOP)

				cRectS.Top = 185
				DrawTextSafe(hdc, T("Enable or disable click sound effects.", "روشن یا خاموش کردن صدای کلیک دکمه‌ها."), &cRectS, DT_LEFT|DT_TOP)

				cRectS.Top = 255
				DrawTextSafe(hdc, T("Run SysGuard automatically when Windows starts.", "اجرای خودکار برنامه هنگام روشن شدن ویندوز."), &cRectS, DT_LEFT|DT_TOP)
			}

			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		}

	case WM_DESTROY:
		user32.NewProc("PostQuitMessage").Call(0)
		return 0

	default:
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), msg, wParam, lParam)
		return ret
	}
	return 0
}

func ShowMessageBox(hwnd syscall.Handle, msg, title string, flags uint32) {
	tPtr, _ := syscall.UTF16PtrFromString(msg)
	cPtr, _ := syscall.UTF16PtrFromString(title)
	procMessageBox.Call(uintptr(hwnd), uintptr(unsafe.Pointer(tPtr)), uintptr(unsafe.Pointer(cPtr)), uintptr(flags))
}

func createAndShowGUI() {
	runtime.LockOSThread()
	hFontNormal = createModernFont(15, 400)
	hFontTitle = createModernFont(26, 700)
	hFontWidget = createModernFont(14, 600)

	inst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("SysGuardClass")
	windowName, _ := syscall.UTF16PtrFromString("SysGuard Ultimate - Modular Edition")
	cursor, _, _ := procLoadCursor.Call(0, uintptr(IDC_ARROW))

	wc := WNDCLASSEX{
		CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc: syscall.NewCallback(wndProc), HInstance: syscall.Handle(inst),
		HCursor: syscall.Handle(cursor), HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		LpszClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwndPtr, _, _ := procCreateWindow.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		uintptr(WS_OVERLAPPEDWINDOW|WS_VISIBLE|WS_CLIPCHILDREN), uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT),
		950, 640, 0, 0, inst, 0)

	hwndMain = syscall.Handle(hwndPtr)
	setWindowIcon(hwndMain, "icon.ico")

	if procDwmSetWindowAttribute.Find() == nil {
		darkMode := int32(1)
		procDwmSetWindowAttribute.Call(uintptr(hwndMain), 20, uintptr(unsafe.Pointer(&darkMode)), uintptr(unsafe.Sizeof(darkMode)))
	}

	hwndTabDash = createControl(hwndMain, IDC_TAB_DASH, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 40, 180, 55)
	hwndTabNet = createControl(hwndMain, IDC_TAB_NET, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 95, 180, 55)

	if isAsusLaptop {
		hwndTabPower = createControl(hwndMain, IDC_TAB_POWER, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 150, 180, 55)
		hwndTabGamepad = createControl(hwndMain, IDC_TAB_GAMEPAD, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 205, 180, 55)
		hwndTabTools = createControl(hwndMain, IDC_TAB_TOOLS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 260, 180, 55)
		// تب جدید در لپ‌تاپ‌های ایسوس
		hwndTabSettings = createControl(hwndMain, IDC_TAB_SETTINGS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 315, 180, 55)

		hwndBtnSilent = createControl(hwndMain, IDC_MODE_SILENT, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 145, 140, 45)
		hwndBtnBalanced = createControl(hwndMain, IDC_MODE_BALANCED, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 385, 145, 140, 45)
		hwndBtnTurbo = createControl(hwndMain, IDC_MODE_TURBO, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 540, 145, 140, 45)

		hwndBtnEco = createControl(hwndMain, IDC_GPU_ECO, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 275, 130, 45)
		hwndBtnStandard = createControl(hwndMain, IDC_GPU_STD, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 375, 275, 130, 45)
		hwndBtnUltra = createControl(hwndMain, IDC_GPU_ULTRA, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 520, 275, 130, 45)
	} else {
		hwndTabGamepad = createControl(hwndMain, IDC_TAB_GAMEPAD, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 150, 180, 55)
		hwndTabTools = createControl(hwndMain, IDC_TAB_TOOLS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 205, 180, 55)
		// تب جدید در سیستم‌های عادی
		hwndTabSettings = createControl(hwndMain, IDC_TAB_SETTINGS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 260, 180, 55)
	}

	// ایجاد دکمه‌های داخل بخش تنظیمات
	hwndBtnLang = createControl(hwndMain, IDC_SET_LANG, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 100, 250, 50)
	hwndBtnSoundToggle = createControl(hwndMain, IDC_SET_SOUND, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 170, 250, 50)
	hwndBtnStartup = createControl(hwndMain, IDC_SET_STARTUP, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 240, 250, 50)

	hwndChkWidget = createControl(hwndMain, IDC_CHK_WIDGET, "BUTTON", " Enable RGB Desktop Overlay", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 210, 535, 230, 30)

	hwndBtnSound = createControl(hwndMain, IDC_BTN_SOUND, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 460, 530, 105, 45)
	hwndBtnDiag = createControl(hwndMain, IDC_BTN_DIAG, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 575, 530, 105, 45)
	hwndBtnAnalyze = createControl(hwndMain, IDC_BTN_ANALYZE, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 690, 530, 105, 45)
	hwndBtnClean = createControl(hwndMain, IDC_BTN_CLEAN, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 805, 530, 105, 45)

	hwndBtnSysInfo = createControl(hwndMain, IDC_BTN_SYSINFO, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 740, 40, 170, 35)

	procSendMessage.Call(uintptr(hwndChkWidget), WM_SETFONT, uintptr(hFontNormal), 1)

	var msg MSG
	for {
		ret, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		procTranslateMsg.Call(uintptr(unsafe.Pointer(&msg)))
		procDispatchMsg.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func createControl(parent syscall.Handle, id uint32, class, text string, style uint32, x, y, width, height int) syscall.Handle {
	inst, _, _ := procGetModuleHandle.Call(0)
	ret, _, _ := procCreateWindow.Call(0, uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(class))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(text))), uintptr(style),
		uintptr(x), uintptr(y), uintptr(width), uintptr(height), uintptr(parent), uintptr(id), inst, 0)
	return syscall.Handle(ret)
}
