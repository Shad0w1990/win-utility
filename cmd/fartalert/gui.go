package main

import (
	"fmt"
	"math"
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
	saveSettings()
}

func playUIClick() {
	if uiSoundEnabled {
		go procBeep.Call(uintptr(700), uintptr(30))
	}
}

func blendColor(c1, c2 uint32, t float64) uint32 {
	r1, g1, b1 := c1&0xFF, (c1>>8)&0xFF, (c1>>16)&0xFF
	r2, g2, b2 := c2&0xFF, (c2>>8)&0xFF, (c2>>16)&0xFF
	r := uint32(float64(r1) + t*(float64(r2)-float64(r1)))
	g := uint32(float64(g1) + t*(float64(g2)-float64(g1)))
	b := uint32(float64(b1) + t*(float64(b2)-float64(b1)))
	return r | (g << 8) | (b << 16)
}

func drawModernProgressBar(hdc uintptr, x, y, width, height, percent int, color uint32) {
	drawRect(hdc, x, y, x+width, y+height, height/2, RGB(40, 48, 60))
	if percent > 0 {
		if percent > 100 {
			percent = 100
		}
		fillWidth := (width * percent) / 100
		drawRect(hdc, x, y, x+fillWidth, y+height, height/2, color)
	}
}

func updateSliderValue(mouseX, startX, width int32, val *int) {
	pct := float64(mouseX-startX) / float64(width) * 100.0
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	*val = int(pct)
}

func updateTabButtonsVisibility() {
	showG := (isAsus && currentPage == 2)
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

	showTools := (isAsus && currentPage == 4) || (!isAsus && currentPage == 3)
	swT := SW_HIDE
	if showTools {
		swT = SW_SHOW
	}

	procShowWindow.Call(uintptr(hwndBtnPwrSave), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnPwrBal), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnPwrHigh), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnSound), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnDiag), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnAnalyze), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnClean), uintptr(swT))
	procShowWindow.Call(uintptr(hwndBtnEventLog), uintptr(swT))

	showS := (isAsus && currentPage == 5) || (!isAsus && currentPage == 4)
	swS := SW_HIDE
	if showS {
		swS = SW_SHOW
	}
	procShowWindow.Call(uintptr(hwndBtnLang), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnSoundToggle), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnStartup), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnCopyIran), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnCopyTron), uintptr(swS))
	procShowWindow.Call(uintptr(hwndBtnCopyTon), uintptr(swS))

	if hwndTabDash != 0 {
		procInvalidateRect.Call(uintptr(hwndTabDash), 0, 1)
	}
	if hwndTabNet != 0 {
		procInvalidateRect.Call(uintptr(hwndTabNet), 0, 1)
	}
	if hwndTabPower != 0 {
		procInvalidateRect.Call(uintptr(hwndTabPower), 0, 1)
	}
	if hwndTabGamepad != 0 {
		procInvalidateRect.Call(uintptr(hwndTabGamepad), 0, 1)
	}
	if hwndTabTools != 0 {
		procInvalidateRect.Call(uintptr(hwndTabTools), 0, 1)
	}
	if hwndTabSettings != 0 {
		procInvalidateRect.Call(uintptr(hwndTabSettings), 0, 1)
	}

	procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
	procUpdateWindow.Call(uintptr(hwndMain))
}

func wndProc(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_ERASEBKGND:
		return 1
	case WM_CREATE:
		loadSettings()

		brushPtr, _, _ := procCreateSolidBrush.Call(uintptr(RGB(10, 14, 25)))
		hBrushBg = syscall.Handle(brushPtr)

		checkStartup()

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

		if (isAsus && currentPage == 3) || (!isAsus && currentPage == 2) {
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
			case IDC_BTN_EVENT_LOG:
				hBtn = hwndBtnEventLog
			case IDC_BTN_SYSINFO:
				hBtn = hwndBtnSysInfo
			case IDC_TOOL_PWR_SAVE:
				hBtn = hwndBtnPwrSave
			case IDC_TOOL_PWR_BAL:
				hBtn = hwndBtnPwrBal
			case IDC_TOOL_PWR_HIGH:
				hBtn = hwndBtnPwrHigh
			case IDC_SET_LANG:
				hBtn = hwndBtnLang
			}
			if hBtn != 0 {
				procInvalidateRect.Call(uintptr(hBtn), 0, 1)
			}
		}

	case WM_LBUTTONDOWN:
		x := int32(lParam & 0xFFFF)
		y := int32((lParam >> 16) & 0xFFFF)

		if (isAsus && currentPage == 4) || (!isAsus && currentPage == 3) {
			if isLaptop && x >= 230 && x <= 420 && y >= 275 && y <= 305 {
				isDraggingBrightness = true
				user32.NewProc("SetCapture").Call(uintptr(hwnd))
				updateSliderValue(x, 230, 190, &sysBrightnessVal)
				rectUpdate := RECT{Left: 210, Top: 270, Right: 490, Bottom: 320}
				procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rectUpdate)), 0)
				return 0
			}
			if x >= 530 && x <= 720 && y >= 275 && y <= 305 {
				isDraggingVolume = true
				user32.NewProc("SetCapture").Call(uintptr(hwnd))
				updateSliderValue(x, 530, 190, &sysVolumeVal)
				rectUpdate := RECT{Left: 510, Top: 270, Right: 790, Bottom: 320}
				procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rectUpdate)), 0)
				return 0
			}
		}

		user32.NewProc("ReleaseCapture").Call()
		procSendMessage.Call(uintptr(hwnd), uintptr(WM_NCLBUTTONDOWN), uintptr(HTCAPTION), 0)
		return 0

	case WM_MOUSEMOVE:
		if isDraggingBrightness || isDraggingVolume {
			x := int32(lParam & 0xFFFF)
			if isDraggingBrightness {
				updateSliderValue(x, 230, 190, &sysBrightnessVal)
				rectUpdate := RECT{Left: 210, Top: 270, Right: 490, Bottom: 320}
				procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rectUpdate)), 0)
			} else if isDraggingVolume {
				updateSliderValue(x, 530, 190, &sysVolumeVal)
				rectUpdate := RECT{Left: 510, Top: 270, Right: 790, Bottom: 320}
				procInvalidateRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rectUpdate)), 0)
			}
			return 0
		}

	case WM_LBUTTONUP:
		if isDraggingBrightness {
			isDraggingBrightness = false
			user32.NewProc("ReleaseCapture").Call()
			saveSettings()
			go func(val int) {
				cmd := exec.Command("powershell", "-NoProfile", "-Command", fmt.Sprintf("(Get-WmiObject -Namespace root/WMI -Class WmiMonitorBrightnessMethods).WmiSetBrightness(1, %d)", val))
				cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				cmd.Run()
			}(sysBrightnessVal)
			return 0
		}
		if isDraggingVolume {
			isDraggingVolume = false
			user32.NewProc("ReleaseCapture").Call()
			saveSettings()
			go func(val int) {
				setSystemVolumeNatively(val)
			}(sysVolumeVal)
			return 0
		}

	case WM_CTLCOLORSTATIC:
		hdc := syscall.Handle(wParam)
		procSetBkMode.Call(uintptr(hdc), uintptr(TRANSPARENT))
		procSetTextColor.Call(uintptr(hdc), uintptr(RGB(180, 200, 220)))
		return uintptr(hBrushBg)

	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis.CtlType == ODT_BUTTON {

			isMaintenanceBtn := (dis.CtlID >= 1001 && dis.CtlID <= 1006) && dis.CtlID != IDC_BTN_SYSINFO
			if isMaintenanceBtn {
				currentLoading := atomic.LoadUint32(&loadingButtonID)
				isDown := (dis.ItemState & ODS_SELECTED) != 0

				var borderColor uint32
				var title, desc string

				switch dis.CtlID {
				case IDC_BTN_SOUND:
					borderColor = RGB(200, 80, 255)
					title = T("🔊 Audio Test", "🔊 تست کارت صدا")
					desc = T("Wake up and test audio drivers.", "ارسال سیگنال برای تست درایور صوتی.")
				case IDC_BTN_DIAG:
					borderColor = RGB(0, 200, 255)
					title = T("🔄 Force Sync", "🔄 همگام‌سازی")
					desc = T("Syncs system clock & flushes DNS.", "تنظیم دقیق ساعت و پاکسازی کش اینترنت.")
				case IDC_BTN_CLEAN:
					borderColor = RGB(255, 140, 50)
					title = T("🧹 Deep OS Clean", "🧹 پاکسازی عمیق")
					desc = T("Clears temp files & frees up memory.", "حذف فایل‌های موقت و آزادسازی حافظه.")
				case IDC_BTN_ANALYZE:
					borderColor = RGB(0, 255, 150)
					title = T("⚙️ Hardware Bottleneck", "⚙️ آنالیز گلوگاه قطعات")
					desc = T("Analyzes 7-day telemetry to find weak components.", "تحلیل دیتای ۷ روزه برای تشخیص قطعه ضعیف سیستم.")
				case IDC_BTN_EVENT_LOG:
					borderColor = RGB(255, 80, 100)
					title = T("🧠 AI Error Scanner", "🧠 هوش مصنوعی سیستم")
					desc = T("Finds hidden OS errors via Gemini AI.", "کشف خطاهای پنهان ویندوز با هوش مصنوعی.")
				}

				fillColor := uint32(RGB(15, 20, 30))
				txtColor := borderColor

				if currentLoading == dis.CtlID {
					rad := globalHue * math.Pi / 180.0
					t := (math.Sin(rad*2.0) + 1.0) / 2.0
					fillColor = blendColor(RGB(15, 20, 30), borderColor, t)
					txtColor = RGB(255, 255, 255)
				} else if isDown {
					fillColor = borderColor
					txtColor = RGB(10, 14, 25)
				}

				drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 14, borderColor)
				if fillColor != borderColor {
					drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 12, fillColor)
				}

				rTitle := RECT{Left: dis.RcItem.Left + 5, Top: dis.RcItem.Top + 14, Right: dis.RcItem.Right - 5, Bottom: dis.RcItem.Bottom}
				procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
				procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
				procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
				DrawTextSafe(uintptr(dis.Hdc), title, &rTitle, DT_CENTER|DT_TOP|DT_SINGLELINE)

				rDesc := RECT{Left: dis.RcItem.Left + 8, Top: dis.RcItem.Top + 38, Right: dis.RcItem.Right - 8, Bottom: dis.RcItem.Bottom}
				procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontSmall))
				descColor := uint32(RGB(140, 150, 160))
				if isDown || currentLoading == dis.CtlID {
					descColor = txtColor
				}
				procSetTextColor.Call(uintptr(dis.Hdc), uintptr(descColor))
				DrawTextSafe(uintptr(dis.Hdc), desc, &rDesc, DT_CENTER|DT_TOP|0x00000010)

				return 1
			}

			if dis.CtlID == IDC_BTN_SYSINFO {
				currentLoading := atomic.LoadUint32(&loadingButtonID)
				isDown := (dis.ItemState & ODS_SELECTED) != 0
				borderColor := RGB(0, 255, 150)
				fillColor := RGB(15, 20, 30)
				txtColor := borderColor
				text := T("📄 Full System Info", "📄 اطلاعات جامع قطعات")

				if currentLoading == dis.CtlID {
					rad := globalHue * math.Pi / 180.0
					t := (math.Sin(rad*2.0) + 1.0) / 2.0
					fillColor = blendColor(RGB(15, 20, 30), borderColor, t)
					txtColor = RGB(255, 255, 255)
				} else if isDown {
					fillColor = borderColor
					txtColor = RGB(10, 14, 25)
				}

				drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 14, borderColor)
				if fillColor != borderColor {
					drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 12, fillColor)
				}

				procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
				procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
				procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
				DrawTextSafe(uintptr(dis.Hdc), text, &dis.RcItem, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
				return 1
			}

			// استایل دکمه‌های کپی دونیت
			isDonateBtn := dis.CtlID == IDC_COPY_IRAN || dis.CtlID == IDC_COPY_TRON || dis.CtlID == IDC_COPY_TON
			if isDonateBtn {
				isDown := (dis.ItemState & ODS_SELECTED) != 0
				var borderColor uint32
				text := ""

				switch dis.CtlID {
				case IDC_COPY_IRAN:
					borderColor = RGB(0, 255, 150)
					text = T("📋 Copy Card", "📋 کپی کارت")
				case IDC_COPY_TRON:
					borderColor = RGB(255, 140, 50)
					text = T("📋 Copy Tron", "📋 کپی ترون")
				case IDC_COPY_TON:
					borderColor = RGB(0, 210, 255)
					text = T("📋 Copy TON", "📋 کپی تون")
				}

				fillColor := uint32(RGB(15, 20, 30))
				txtColor := borderColor

				if isDown {
					fillColor = borderColor
					txtColor = RGB(10, 14, 25)
				}

				drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 10, borderColor)
				if fillColor != borderColor {
					drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 8, fillColor)
				}

				procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontWidget))
				procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
				procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
				DrawTextSafe(uintptr(dis.Hdc), text, &dis.RcItem, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
				return 1
			}

			drawCustomButton(dis)
			return 1
		}

	case WM_COMMAND:
		if int(wParam>>16) == BN_CLICKED {
			playUIClick()
			cmdID := int(wParam & 0xFFFF)
			switch cmdID {
			case IDC_TAB_DASH:
				currentPage = 0
				updateTabButtonsVisibility()
			case IDC_TAB_NET:
				currentPage = 1
				updateTabButtonsVisibility()
			case IDC_TAB_POWER:
				if isAsus {
					currentPage = 2
					updateTabButtonsVisibility()
				}
			case IDC_TAB_GAMEPAD:
				if isAsus {
					currentPage = 3
				} else {
					currentPage = 2
				}
				updateTabButtonsVisibility()
			case IDC_TAB_TOOLS:
				if isAsus {
					currentPage = 4
				} else {
					currentPage = 3
				}
				updateTabButtonsVisibility()
			case IDC_TAB_SETTINGS:
				if isAsus {
					currentPage = 5
				} else {
					currentPage = 4
				}
				updateTabButtonsVisibility()

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
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_SET_LANG)
					procInvalidateRect.Call(uintptr(hwndBtnLang), 0, 1)
					go func() {
						time.Sleep(500 * time.Millisecond)
						if uiLanguage == "EN" {
							uiLanguage = "FA"
						} else {
							uiLanguage = "EN"
						}
						saveSettings()
						atomic.StoreUint32(&loadingButtonID, 0)
						chkText, _ := syscall.UTF16PtrFromString(T(" Enable RGB Desktop Overlay", " فعال‌سازی ویجت دسکتاپ (RGB)"))
						procSetWindowText.Call(uintptr(hwndChkWidget), uintptr(unsafe.Pointer(chkText)))
						updateTabButtonsVisibility()
					}()
				}

			case IDC_SET_SOUND:
				uiSoundEnabled = !uiSoundEnabled
				saveSettings()
				procInvalidateRect.Call(uintptr(hwndBtnSoundToggle), 0, 1)
			case IDC_SET_STARTUP:
				toggleStartup()
				procInvalidateRect.Call(uintptr(hwndBtnStartup), 0, 1)

			// عملیات کپی دونیت‌ها با شماره کارت جدید شما
			case IDC_COPY_IRAN:
				copyToClipboardNative("شماره کارت: 6104-3377-6761-3068\nبه نام: احسان خرسند")
				ShowMessageBox(hwndMain, T("Iranian card number copied to clipboard.", "شماره کارت ریالی با موفقیت کپی شد."), "Copied", 0x00000040)
			case IDC_COPY_TRON:
				copyToClipboardNative("TCzZtuWEwZfcWa3wKHGYjwHZrSPL6DW7C")
				ShowMessageBox(hwndMain, T("USDT (TRC20) address copied to clipboard.", "آدرس تتر شبکه Tron (TRC20) کپی شد."), "Copied", 0x00000040)
			case IDC_COPY_TON:
				copyToClipboardNative("UQB-5yLspFNXmvEXR4DP955To-D3hn2b0Rc3p7BNCqfzZAtF")
				ShowMessageBox(hwndMain, T("TON address copied to clipboard.", "آدرس شبکه TON کپی شد."), "Copied", 0x00000040)

			case IDC_TOOL_PWR_SAVE:
				if strings.EqualFold(sysPowerPlanGUID, "a1841308-3541-4fab-bc81-f71556f20b4a") {
					break
				}
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_TOOL_PWR_SAVE)
					sysPowerPlanGUID = "loading_state"
					procInvalidateRect.Call(uintptr(hwndBtnPwrSave), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrBal), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrHigh), 0, 1)
					go func() {
						cmd := exec.Command("powercfg", "/setactive", "a1841308-3541-4fab-bc81-f71556f20b4a")
						cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						cmd.Run()
						time.Sleep(600 * time.Millisecond)
						updatePowerPlanStatus()
						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
					}()
				}
			case IDC_TOOL_PWR_BAL:
				if strings.EqualFold(sysPowerPlanGUID, "381b4222-f694-41f0-9685-ff5bb260df2e") {
					break
				}
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_TOOL_PWR_BAL)
					sysPowerPlanGUID = "loading_state"
					procInvalidateRect.Call(uintptr(hwndBtnPwrSave), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrBal), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrHigh), 0, 1)
					go func() {
						cmd := exec.Command("powercfg", "/setactive", "381b4222-f694-41f0-9685-ff5bb260df2e")
						cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						cmd.Run()
						time.Sleep(600 * time.Millisecond)
						updatePowerPlanStatus()
						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
					}()
				}
			case IDC_TOOL_PWR_HIGH:
				if strings.EqualFold(sysPowerPlanGUID, "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c") {
					break
				}
				if atomic.LoadUint32(&loadingButtonID) == 0 {
					atomic.StoreUint32(&loadingButtonID, IDC_TOOL_PWR_HIGH)
					sysPowerPlanGUID = "loading_state"
					procInvalidateRect.Call(uintptr(hwndBtnPwrSave), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrBal), 0, 1)
					procInvalidateRect.Call(uintptr(hwndBtnPwrHigh), 0, 1)
					go func() {
						cmd := exec.Command("powercfg", "/setactive", "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c")
						cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
						cmd.Run()
						time.Sleep(600 * time.Millisecond)
						updatePowerPlanStatus()
						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
					}()
				}

			case IDC_MODE_SILENT:
				currentPowerMode = "Silent"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_MODE_BALANCED:
				currentPowerMode = "Balanced"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_MODE_TURBO:
				currentPowerMode = "Turbo"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_ECO:
				currentGpuMode = "Eco"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_STD:
				currentGpuMode = "Standard"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)
			case IDC_GPU_ULTRA:
				currentGpuMode = "Ultimate"
				saveSettings()
				procInvalidateRect.Call(uintptr(hwnd), 0, 1)

			case IDC_BTN_SYSINFO:
				OpenSysInfoWindow()

			case IDC_BTN_EVENT_LOG:
				OpenAnalyzerWindow()

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
						cReason := fmt.Sprintf(T("Avg Load: %.1f%% - ", "فشار پردازشی: %.1f%% - "), cAvg)
						if cAvg > 85 {
							cScore += 30
							cReason += T("CRITICAL: Processor is bottlenecking.", "بحرانی: پردازنده باعث افت فریم می‌شود.")
						} else if cAvg > 60 {
							cReason += T("Moderate load. Good for now.", "فشار متوسط است. در حال حاضر مشکلی ندارد.")
						} else {
							cReason += T("Excellent headroom.", "عالی. پردازنده قدرت کافی دارد.")
						}
						comps = append(comps, Component{T("Processor (CPU)", "پردازنده (CPU)"), cScore, cReason})

						rScore := rAvg
						rReason := fmt.Sprintf(T("Avg Usage: %.1f%% - ", "مصرف: %.1f%% - "), rAvg)
						if rAvg > 85 {
							rScore += 40
							rReason += T("CRITICAL: Upgrade RAM immediately.", "بحرانی: ظرفیت رم در حال پر شدن است.")
						} else if totalRamGB <= 8 && rAvg > 60 {
							rScore += 20
							rReason += T("Consider 16GB+ upgrade.", "ارتقا رم به 16 گیگابایت پیشنهاد می‌شود.")
						} else {
							rReason += T("Memory capacity is sufficient.", "ظرفیت رم کاملاً جوابگو است.")
						}
						comps = append(comps, Component{T("Memory (RAM)", "رم (RAM)"), rScore, rReason})

						gScore := gAvg
						gReason := fmt.Sprintf(T("Avg Load: %.1f%% - ", "فشار پردازشی: %.1f%% - "), gAvg)
						if gAvg > 90 {
							gScore += 25
							gReason += T("High load. Upgrade if gaming lags.", "فشار بالا. اگر لگ دارید گرافیک را ارتقا دهید.")
						} else {
							gReason += T("Performance is optimal.", "کارت گرافیک در وضعیت ایده‌آل است.")
						}
						comps = append(comps, Component{T("Graphics (GPU)", "گرافیک (GPU)"), gScore, gReason})

						dScore := dAvg
						dReason := fmt.Sprintf(T("Space Used: %.1f%% - ", "فضای پر شده: %.1f%% - "), dAvg)
						if dAvg > 90 {
							dScore += 45
							dReason += T("CRITICAL: Drive almost full!", "بحرانی: درایو ویندوز در حال پر شدن است.")
						} else if strings.Contains(strings.ToUpper(diskMediaType), "HDD") {
							dScore += 60
							dReason += T("CRITICAL: Upgrade to SSD.", "بحرانی: ویندوز روی هارد دیسک است. خرید SSD ضروری است.")
						} else {
							dReason += T("Storage health is good.", "وضعیت درایوها عالی است.")
						}
						comps = append(comps, Component{T("Storage (Disk)", "حافظه (Disk)"), dScore, dReason})

						sort.Slice(comps, func(i, j int) bool { return comps[i].Score > comps[j].Score })

						report := T("🧠 Smart Hardware Analysis\n", "🧠 گزارش تحلیل سخت‌افزار (Bottleneck)\n")
						if count > 0 {
							report += fmt.Sprintf(T("Based on %d historic data points.\n\n", "بر اساس %d نقطه داده تاریخی.\n\n"), int(count))
						} else {
							report += T("Based on current real-time snapshot.\n\n", "بر اساس اسکن لحظه‌ای (زنده).\n\n")
						}

						report += T("⚠️ UPGRADE PRIORITY RANKING:\n\n", "⚠️ اولویت ارتقاء قطعات (شماره ۱ ضعیف‌ترین):\n\n")
						for i, c := range comps {
							report += fmt.Sprintf(T("Rank %d: %s\n   └ %s\n\n", "رتبه %d: %s\n   └ %s\n\n"), i+1, c.Name, c.Reason)
						}

						atomic.StoreUint32(&loadingButtonID, 0)
						procInvalidateRect.Call(uintptr(hwndBtnAnalyze), 0, 1)
						ShowMessageBox(hwndMain, report, T("Hardware Analysis", "آنالیز سخت‌افزار"), 0x00000040)
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

				panelColor := RGB(16, 22, 35)

				ipBadgeLeft := int(rect.Right) - 230
				drawRect(hdc, ipBadgeLeft, 33, int(rect.Right)-30, 67, 16, RGB(0, 210, 255))
				drawRect(hdc, ipBadgeLeft+1, 34, int(rect.Right)-31, 66, 14, panelColor)

				procSelectObject.Call(hdc, uintptr(hFontNormal))
				setTextColor(hdc, RGB(0, 255, 150))
				rIP := RECT{Left: int32(ipBadgeLeft), Top: 41, Right: rect.Right - 30, Bottom: 70}
				DrawTextSafe(hdc, "🌐 "+sysIPAddress, &rIP, DT_CENTER|DT_TOP)

				drawRect(hdc, 210, 90, int(rect.Right)-30, 145, 10, panelColor)
				drawRect(hdc, 210, 155, int(rect.Right)-30, 245, 10, panelColor)

				statusColor := RGB(255, 120, 120)
				if strings.Contains(netUpdateText, "RUNNING") {
					statusColor = RGB(255, 180, 0)
				} else if strings.Contains(netUpdateText, "STOPPED") {
					statusColor = RGB(0, 255, 150)
				}
				setTextColor(hdc, statusColor)
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

			} else if isAsus && currentPage == 2 {
				DrawTextSafe(hdc, T("G-HELPER CONTROL PANEL", "کنترل پنل G-HELPER"), &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)

				drawRect(hdc, 210, 95, int(rect.Right)-30, 210, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				contentRect.Top, contentRect.Left = 110, 230
				DrawTextSafe(hdc, fmt.Sprintf(T("⚡ Operating Mode: %s", "⚡ پروفایل عملکرد: %s"), currentPowerMode), &contentRect, DT_LEFT|DT_TOP)

				drawRect(hdc, 210, 225, int(rect.Right)-30, 340, 12, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				contentRect.Top = 240
				DrawTextSafe(hdc, fmt.Sprintf(T("🎮 GPU Ultimate Control: %s", "🎮 کنترل مستقیم گرافیک: %s"), currentGpuMode), &contentRect, DT_LEFT|DT_TOP)

				drawRect(hdc, 210, 355, int(rect.Right)-30, 465, 12, panelColor)
				setTextColor(hdc, RGB(255, 180, 0))
				contentRect.Top = 370
				DrawTextSafe(hdc, fmt.Sprintf(T("🔋 Battery Charge Limit: %s (Current: %s)", "🔋 محدودیت شارژ باتری: %s (فعلی: %s)"), batteryLimit, batteryLevel), &contentRect, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(180, 200, 220))
				contentRect.Top = 405
				DrawTextSafe(hdc, T("Protecting battery lifespan with automated charge capping.", "محافظت از طول عمر باتری با کنترل هوشمند شارژ."), &contentRect, DT_LEFT|DT_TOP)

			} else if (isAsus && currentPage == 3) || (!isAsus && currentPage == 2) {

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

			} else if (isAsus && currentPage == 4) || (!isAsus && currentPage == 3) {
				DrawTextSafe(hdc, T("SYSTEM TOOLS & STATUS", "ابزارهای سیستمی و وضعیت‌ها"), &contentRect, DT_LEFT|DT_TOP)
				procSelectObject.Call(hdc, uintptr(hFontNormal))

				panelColor := RGB(16, 22, 35)

				drawRect(hdc, 210, 90, 490, 215, 12, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				cRect1 := RECT{Left: 230, Top: 110, Right: 470, Bottom: 190}

				if isLaptop {
					DrawTextSafe(hdc, T("🔋 Battery Status", "🔋 وضعیت باتری"), &cRect1, DT_LEFT|DT_TOP)
					setTextColor(hdc, RGB(220, 235, 255))
					cRect1.Top += 35
					DrawTextSafe(hdc, sysBatteryPercent, &cRect1, DT_LEFT|DT_TOP)
				} else {
					DrawTextSafe(hdc, T("🔌 Power & Uptime", "🔌 منبع تغذیه و آپتایم"), &cRect1, DT_LEFT|DT_TOP)
					setTextColor(hdc, RGB(220, 235, 255))
					cRect1.Top += 35
					DrawTextSafe(hdc, fmt.Sprintf("AC Power | Uptime: %s", sysUptime), &cRect1, DT_LEFT|DT_TOP)
				}

				drawRect(hdc, 510, 90, 790, 215, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				cRect2 := RECT{Left: 530, Top: 110, Right: 770, Bottom: 190}
				DrawTextSafe(hdc, T("⚡ Power Plan", "⚡ مصرف انرژی"), &cRect2, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				cRect2.Top += 35

				if atomic.LoadUint32(&loadingButtonID) >= IDC_TOOL_PWR_SAVE && atomic.LoadUint32(&loadingButtonID) <= IDC_TOOL_PWR_HIGH {
					DrawTextSafe(hdc, T("Applying profile...", "در حال اعمال تنظیمات..."), &cRect2, DT_LEFT|DT_TOP)
				} else {
					DrawTextSafe(hdc, sysPowerPlan, &cRect2, DT_LEFT|DT_TOP)
				}

				drawRect(hdc, 210, 225, 490, 350, 12, panelColor)
				setTextColor(hdc, RGB(255, 180, 0))
				cRect3 := RECT{Left: 230, Top: 245, Right: 470, Bottom: 320}

				if isLaptop {
					DrawTextSafe(hdc, T("☀️ Display Brightness", "☀️ روشنایی مانیتور"), &cRect3, DT_LEFT|DT_TOP)
					drawModernProgressBar(hdc, 230, 292, 175, 10, sysBrightnessVal, RGB(255, 180, 0))
					setTextColor(hdc, RGB(220, 235, 255))
					cRect3Val := RECT{Left: 415, Top: 287, Right: 480, Bottom: 310}
					DrawTextSafe(hdc, fmt.Sprintf("%d%%", sysBrightnessVal), &cRect3Val, DT_LEFT|DT_VCENTER|DT_SINGLELINE)
				} else {
					DrawTextSafe(hdc, T("🖥️ Display Output", "🖥️ خروجی تصویر"), &cRect3, DT_LEFT|DT_TOP)
					setTextColor(hdc, RGB(220, 235, 255))
					cRect3.Top += 35
					DrawTextSafe(hdc, "External Monitor (DDC/CI)", &cRect3, DT_LEFT|DT_TOP)
				}

				drawRect(hdc, 510, 225, 790, 350, 12, panelColor)
				setTextColor(hdc, RGB(200, 80, 255))
				cRect4 := RECT{Left: 530, Top: 245, Right: 770, Bottom: 320}
				DrawTextSafe(hdc, T("🔊 System Volume", "🔊 وضعیت صدا"), &cRect4, DT_LEFT|DT_TOP)
				drawModernProgressBar(hdc, 530, 292, 175, 10, sysVolumeVal, RGB(200, 80, 255))
				setTextColor(hdc, RGB(220, 235, 255))
				cRect4Val := RECT{Left: 715, Top: 287, Right: 780, Bottom: 310}
				DrawTextSafe(hdc, fmt.Sprintf("%d%%", sysVolumeVal), &cRect4Val, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

				drawRect(hdc, 210, 365, 790, 580, 12, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				cRect5 := RECT{Left: 230, Top: 380, Right: 770, Bottom: 420}
				DrawTextSafe(hdc, T("🛠️ System Maintenance & AI Diagnostics", "🛠️ ابزارهای نگهداری و هوش مصنوعی"), &cRect5, DT_LEFT|DT_TOP)

			} else if (isAsus && currentPage == 5) || (!isAsus && currentPage == 4) {
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

				setTextColor(hdc, RGB(0, 210, 255))
				cRectD := RECT{Left: 210, Top: 340, Right: rect.Right - 30, Bottom: 370}
				DrawTextSafe(hdc, T("☕ SUPPORT THE DEVELOPER", "☕ حمایت از توسعه‌دهنده (Donate)"), &cRectD, DT_LEFT|DT_TOP)

				// پنل شبکه شتاب (ریالی)
				drawRect(hdc, 210, 380, int(rect.Right)-30, 445, 10, panelColor)
				setTextColor(hdc, RGB(0, 255, 150))
				rIrTitle := RECT{Left: 230, Top: 393, Right: 500, Bottom: 415}
				DrawTextSafe(hdc, T("💳 Iranian Users (Shetab)", "💳 کاربران داخل ایران (شبکه شتاب)"), &rIrTitle, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rIrNum := RECT{Left: 230, Top: 416, Right: 600, Bottom: 438}
				DrawTextSafe(hdc, "6104-3377-6761-3068  |  Ehsan Khorsand", &rIrNum, DT_LEFT|DT_TOP)

				// پنل تتر Tron (TRC20)
				drawRect(hdc, 210, 455, int(rect.Right)-30, 520, 10, panelColor)
				setTextColor(hdc, RGB(255, 140, 50))
				rTrTitle := RECT{Left: 230, Top: 468, Right: 500, Bottom: 490}
				DrawTextSafe(hdc, T("🌐 USDT - Tron (TRC20)", "🌐 تتر - شبکه ترون (TRC20)"), &rTrTitle, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rTrNum := RECT{Left: 230, Top: 491, Right: 600, Bottom: 513}
				DrawTextSafe(hdc, "TCzZtuWEwZfcWa3wKHGYjwHZrSPL6DW7C", &rTrNum, DT_LEFT|DT_TOP)

				// پنل شبکه TON
				drawRect(hdc, 210, 530, int(rect.Right)-30, 595, 10, panelColor)
				setTextColor(hdc, RGB(0, 210, 255))
				rTonTitle := RECT{Left: 230, Top: 543, Right: 500, Bottom: 565}
				DrawTextSafe(hdc, T("💎 TON Network", "💎 شبکه تون (TON)"), &rTonTitle, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rTonNum := RECT{Left: 230, Top: 566, Right: 600, Bottom: 588}
				DrawTextSafe(hdc, "UQB-5yLspFNXmvEXR4DP955To-D3hn2b0Rc3p7BNCqfzZAtF", &rTonNum, DT_LEFT|DT_TOP)
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
	hFontSmall = createModernFont(12, 400)

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

	if isAsus {
		hwndTabPower = createControl(hwndMain, IDC_TAB_POWER, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 150, 180, 55)
		hwndTabGamepad = createControl(hwndMain, IDC_TAB_GAMEPAD, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 205, 180, 55)
		hwndTabTools = createControl(hwndMain, IDC_TAB_TOOLS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 260, 180, 55)
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
		hwndTabSettings = createControl(hwndMain, IDC_TAB_SETTINGS, "BUTTON", "", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, 0, 260, 180, 55)
	}

	hwndBtnLang = createControl(hwndMain, IDC_SET_LANG, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 100, 250, 50)
	hwndBtnSoundToggle = createControl(hwndMain, IDC_SET_SOUND, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 170, 250, 50)
	hwndBtnStartup = createControl(hwndMain, IDC_SET_STARTUP, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 240, 250, 50)

	// ۳ دکمه کپی برای بخش حمایت مالی (ریالی، ترون، تون)
	hwndBtnCopyIran = createControl(hwndMain, IDC_COPY_IRAN, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 630, 385, 140, 35)
	hwndBtnCopyTron = createControl(hwndMain, IDC_COPY_TRON, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 630, 460, 140, 35)
	hwndBtnCopyTon = createControl(hwndMain, IDC_COPY_TON, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 630, 535, 140, 35)

	hwndBtnPwrSave = createControl(hwndMain, IDC_TOOL_PWR_SAVE, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 520, 165, 80, 35)
	hwndBtnPwrBal = createControl(hwndMain, IDC_TOOL_PWR_BAL, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 610, 165, 80, 35)
	hwndBtnPwrHigh = createControl(hwndMain, IDC_TOOL_PWR_HIGH, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 700, 165, 80, 35)

	hwndBtnSound = createControl(hwndMain, IDC_BTN_SOUND, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 410, 175, 70)
	hwndBtnDiag = createControl(hwndMain, IDC_BTN_DIAG, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 415, 410, 175, 70)
	hwndBtnClean = createControl(hwndMain, IDC_BTN_CLEAN, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 600, 410, 175, 70)
	hwndBtnAnalyze = createControl(hwndMain, IDC_BTN_ANALYZE, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 230, 490, 265, 70)
	hwndBtnEventLog = createControl(hwndMain, IDC_BTN_EVENT_LOG, "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 510, 490, 265, 70)

	hwndChkWidget = createControl(hwndMain, IDC_CHK_WIDGET, "BUTTON", " Enable RGB Desktop Overlay", WS_CHILD|WS_VISIBLE|WS_TABSTOP|BS_AUTOCHECKBOX, 210, 595, 230, 30)
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

func drawCustomButton(dis *DRAWITEMSTRUCT) {
	isTab := (dis.CtlID >= 2001 && dis.CtlID <= 2006)
	isDown := (dis.ItemState & ODS_SELECTED) != 0
	isActiveTab := false

	if dis.CtlID == IDC_TAB_DASH && currentPage == 0 {
		isActiveTab = true
	}
	if dis.CtlID == IDC_TAB_NET && currentPage == 1 {
		isActiveTab = true
	}
	if isAsus {
		if dis.CtlID == IDC_TAB_POWER && currentPage == 2 {
			isActiveTab = true
		}
		if dis.CtlID == IDC_TAB_GAMEPAD && currentPage == 3 {
			isActiveTab = true
		}
		if dis.CtlID == IDC_TAB_TOOLS && currentPage == 4 {
			isActiveTab = true
		}
		if dis.CtlID == IDC_TAB_SETTINGS && currentPage == 5 {
			isActiveTab = true
		}
	} else {
		if dis.CtlID == IDC_TAB_GAMEPAD && currentPage == 2 {
			isActiveTab = true
		}
		if dis.CtlID == IDC_TAB_TOOLS && currentPage == 3 {
			isActiveTab = true
		}
		if dis.CtlID == IDC_TAB_SETTINGS && currentPage == 4 {
			isActiveTab = true
		}
	}

	var text string

	if isTab {
		var bgColor, txtColor uint32
		if isDown {
			bgColor, txtColor = RGB(16, 28, 50), RGB(0, 255, 200)
		} else if isActiveTab {
			bgColor, txtColor = RGB(16, 22, 35), RGB(0, 230, 255)
		} else {
			bgColor, txtColor = RGB(5, 7, 12), RGB(100, 115, 140)
		}

		switch dis.CtlID {
		case IDC_TAB_DASH:
			text = T("    📊  Hardware", "    📊  سخت‌افزار")
		case IDC_TAB_NET:
			text = T("    🌐  Network", "    🌐  شـبکـه")
		case IDC_TAB_POWER:
			text = T("    ⚡  G-Helper", "    ⚡  جی‌-هلپر")
		case IDC_TAB_GAMEPAD:
			text = T("    🎮  Gamepad", "    🎮  دسته بازی")
		case IDC_TAB_TOOLS:
			text = T("    🧰  Tools", "    🧰  ابـزارها")
		case IDC_TAB_SETTINGS:
			text = T("    ⚙️  Settings", "    ⚙️  تنظیمات")
		}

		drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 0, bgColor)
		if isActiveTab {
			drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Left)+5, int(dis.RcItem.Bottom), 0, RGB(0, 210, 255))
		}
		procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
		procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
		procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
		DrawTextSafe(uintptr(dis.Hdc), text, &dis.RcItem, DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	} else {
		currentLoading := atomic.LoadUint32(&loadingButtonID)
		var borderColor, fillColor, txtColor uint32
		var title, desc string
		isActiveMode := false

		isMaintenanceBtn := (dis.CtlID >= 1001 && dis.CtlID <= 1006) && dis.CtlID != IDC_BTN_SYSINFO

		switch dis.CtlID {
		case IDC_BTN_SOUND:
			borderColor = RGB(200, 80, 255)
			title = T("🔊 Audio Test", "🔊 تست کارت صدا")
			desc = T("Wake up and test audio drivers.", "ارسال سیگنال برای تست درایور صوتی.")
		case IDC_BTN_DIAG:
			borderColor = RGB(0, 200, 255)
			title = T("🔄 Force Sync", "🔄 همگام‌سازی")
			desc = T("Syncs system clock & flushes DNS.", "تنظیم دقیق ساعت و پاکسازی کش اینترنت.")
		case IDC_BTN_CLEAN:
			borderColor = RGB(255, 140, 50)
			title = T("🧹 Deep OS Clean", "🧹 پاکسازی عمیق")
			desc = T("Clears temp files & frees up memory.", "حذف فایل‌های موقت و آزادسازی حافظه.")
		case IDC_BTN_ANALYZE:
			borderColor = RGB(0, 255, 150)
			title = T("⚙️ Hardware Bottleneck", "⚙️ آنالیز گلوگاه قطعات")
			desc = T("Analyzes 7-day telemetry to find weak components.", "تحلیل دیتای ۷ روزه برای تشخیص قطعه ضعیف سیستم.")
		case IDC_BTN_EVENT_LOG:
			borderColor = RGB(255, 80, 100)
			title = T("🧠 AI Error Scanner", "🧠 هوش مصنوعی سیستم")
			desc = T("Finds hidden OS errors via Gemini AI.", "کشف خطاهای پنهان ویندوز با هوش مصنوعی.")

		case IDC_MODE_SILENT:
			borderColor = RGB(100, 200, 255)
			text = T("🍃 Silent", "🍃 بی‌صدا")
			if currentPowerMode == "Silent" {
				isActiveMode = true
			}
		case IDC_MODE_BALANCED:
			borderColor = RGB(0, 210, 255)
			text = T("🚗 Balanced", "🚗 متعادل")
			if currentPowerMode == "Balanced" {
				isActiveMode = true
			}
		case IDC_MODE_TURBO:
			borderColor = RGB(255, 80, 100)
			text = T("🚀 Turbo", "🚀 توربو")
			if currentPowerMode == "Turbo" {
				isActiveMode = true
			}
		case IDC_GPU_ECO:
			borderColor = RGB(0, 255, 150)
			text = T("🌱 Eco", "🌱 اقتصادی")
			if currentGpuMode == "Eco" {
				isActiveMode = true
			}
		case IDC_GPU_STD:
			borderColor = RGB(0, 210, 255)
			text = T("🌸 Standard", "🌸 استاندارد")
			if currentGpuMode == "Standard" {
				isActiveMode = true
			}
		case IDC_GPU_ULTRA:
			borderColor = RGB(200, 100, 255)
			text = T("⚡ Ultimate", "⚡ نهایت قدرت")
			if currentGpuMode == "Ultimate" {
				isActiveMode = true
			}

		case IDC_TOOL_PWR_SAVE:
			borderColor = RGB(0, 255, 150)
			text = T("🌱 Save", "🌱 بهینه")
			if strings.EqualFold(sysPowerPlanGUID, "a1841308-3541-4fab-bc81-f71556f20b4a") {
				isActiveMode = true
			}
		case IDC_TOOL_PWR_BAL:
			borderColor = RGB(0, 210, 255)
			text = T("⚖️ Bal", "⚖️ متعادل")
			if strings.EqualFold(sysPowerPlanGUID, "381b4222-f694-41f0-9685-ff5bb260df2e") {
				isActiveMode = true
			}
		case IDC_TOOL_PWR_HIGH:
			borderColor = RGB(255, 80, 100)
			text = T("🔥 High", "🔥 پرقدرت")
			if strings.EqualFold(sysPowerPlanGUID, "8c5e7fda-e8bf-4a96-9a85-a6e23a8c635c") {
				isActiveMode = true
			}

		case IDC_SET_LANG:
			if uiLanguage == "EN" {
				text = "🌐 Language: English"
			} else {
				text = "🌐 زبان نرم افزار: فارسی"
			}
			borderColor = RGB(0, 210, 255)
		case IDC_SET_SOUND:
			if uiSoundEnabled {
				text = T("🔊 UI Click Sounds: ON", "🔊 صدای کلیک دکمه‌ها: روشن")
				borderColor = RGB(0, 255, 150)
				isActiveMode = true
			} else {
				text = T("🔇 UI Click Sounds: OFF", "🔇 صدای کلیک دکمه‌ها: خاموش")
				borderColor = RGB(255, 80, 100)
			}
		case IDC_SET_STARTUP:
			if runAtStartup {
				text = T("🚀 Run on Startup: ON", "🚀 اجرای خودکار با ویندوز: روشن")
				borderColor = RGB(0, 255, 150)
				isActiveMode = true
			} else {
				text = T("🚀 Run on Startup: OFF", "🚀 اجرای خودکار با ویندوز: خاموش")
				borderColor = RGB(255, 80, 100)
			}
		}

		if currentLoading == dis.CtlID {
			rad := globalHue * math.Pi / 180.0
			t := (math.Sin(rad*2.0) + 1.0) / 2.0
			if borderColor == 0 {
				borderColor = RGB(0, 210, 255)
			}
			fillColor = blendColor(RGB(15, 20, 30), borderColor, t)
			txtColor = RGB(255, 255, 255)
		} else if isActiveMode {
			fillColor = borderColor
			txtColor = RGB(10, 14, 25)
		} else {
			isModeButton := (dis.CtlID >= 4001 && dis.CtlID <= 4006) || (dis.CtlID >= 6001 && dis.CtlID <= 6003) || dis.CtlID == IDC_SET_SOUND || dis.CtlID == IDC_SET_STARTUP

			if isModeButton && !isDown {
				borderColor = RGB(50, 60, 75)
				txtColor = RGB(140, 150, 160)
				fillColor = RGB(15, 20, 30)
			} else if isDown {
				fillColor = borderColor
				txtColor = RGB(10, 14, 25)
			} else {
				fillColor = RGB(15, 20, 30)
				txtColor = borderColor
			}
		}

		drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 14, borderColor)
		if fillColor != borderColor {
			drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 12, fillColor)
		}

		if isMaintenanceBtn {
			rTitle := RECT{Left: dis.RcItem.Left + 5, Top: dis.RcItem.Top + 12, Right: dis.RcItem.Right - 5, Bottom: dis.RcItem.Bottom}
			procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
			procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
			procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
			DrawTextSafe(uintptr(dis.Hdc), title, &rTitle, DT_CENTER|DT_TOP|DT_SINGLELINE)

			rDesc := RECT{Left: dis.RcItem.Left + 8, Top: dis.RcItem.Top + 35, Right: dis.RcItem.Right - 8, Bottom: dis.RcItem.Bottom}
			procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontSmall))
			descColor := uint32(RGB(140, 150, 160))
			if isDown || currentLoading == dis.CtlID {
				descColor = txtColor
			}
			procSetTextColor.Call(uintptr(dis.Hdc), uintptr(descColor))
			DrawTextSafe(uintptr(dis.Hdc), desc, &rDesc, DT_CENTER|DT_TOP|0x00000010)
		} else {
			procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontNormal))
			procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
			procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txtColor))
			DrawTextSafe(uintptr(dis.Hdc), text, &dis.RcItem, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
		}
	}
}

func DrawTextSafe(hdc uintptr, text string, rect *RECT, format uint32) {
	if utf16, err := syscall.UTF16FromString(text); err == nil {
		procDrawText.Call(hdc, uintptr(unsafe.Pointer(&utf16[0])), uintptr(len(utf16)-1), uintptr(unsafe.Pointer(rect)), uintptr(format))
	}
}
