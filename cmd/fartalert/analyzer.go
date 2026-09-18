package main

import (
	"fmt"
	"os/exec"
	"strings"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// ساختار لاگ‌ها با اضافه شدن سطح خطا (Level)
type SystemError struct {
	EventID string
	Source  string
	Count   string
	Message string
	Level   string
}

var (
	analyzerErrors    []SystemError
	hwndAnalyzer      syscall.Handle
	hwndAskBtns       [5]syscall.Handle
	hwndCopyBtns      [5]syscall.Handle
	isAnalyzerLoading uint32 = 0
)

const (
	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002
)

// کپی متن با پشتیبانی 100 درصدی از یونیکد (فارسی)
func copyToClipboardNative(text string) {
	utf16, err := syscall.UTF16FromString(text)
	if err != nil {
		return
	}

	procOpenClipboard.Call(0)
	defer procCloseClipboard.Call()
	procEmptyClipboard.Call()

	size := uintptr(len(utf16) * 2)
	hMem, _, _ := procGlobalAlloc.Call(uintptr(GMEM_MOVEABLE), size)
	if hMem == 0 {
		return
	}

	ptr, _, _ := procGlobalLock.Call(hMem)
	if ptr != 0 {
		var src = unsafe.Slice((*uint16)(unsafe.Pointer(&utf16[0])), len(utf16))
		var dst = unsafe.Slice((*uint16)(unsafe.Pointer(ptr)), len(utf16))
		copy(dst, src)
		procGlobalUnlock.Call(hMem)
	}

	procSetClipboardData.Call(uintptr(CF_UNICODETEXT), hMem)
}

func openGemini() {
	cmd := exec.Command("cmd", "/c", "start", "https://gemini.google.com")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Run()
}

// تولید پرامپت حرفه‌ای با توجه به زبانِ انتخاب شده در برنامه
func generateAIPrompt(errData SystemError) string {
	if uiLanguage == "FA" {
		return fmt.Sprintf(`نقش شما: متخصص ارشد ویندوز و عیب‌یابی سیستم (System Administrator)
مشکل: سیستم ویندوزی من در ۷ روز گذشته خطای زیر را ثبت کرده است.

جزئیات خطا:
- سطح خطا (Severity): %s
- منبع (Source): %s
- کد رویداد (Event ID): %s
- دفعات تکرار در یک هفته: %s بار

نمونه متن اورجینال ثبت شده توسط ویندوز:
"%s"

درخواست:
لطفاً پاسخ را فقط به زبان فارسی و با لحن حرفه‌ای در قالب زیر به من ارائه بده:
۱. علت اصلی: (حداکثر ۲ خط ساده و دقیق توضیح بده چرا این اتفاق افتاده)
۲. میزان جدیت: (آیا خطرناک است و نیاز به بررسی فوری دارد یا نویز بی‌خطر است؟)
۳. راه‌حل‌های قدم به قدم: (۳ راهکار عملی، دقیق و مرحله‌به‌مرحله برای رفع دائمی. در صورت نیاز کدهای CMD/PowerShell را دقیق بنویس)`,
			errData.Level, errData.Source, errData.EventID, errData.Count, errData.Message)
	}

	// پرامپت در صورتی که زبان برنامه انگلیسی باشد
	return fmt.Sprintf(`Role: Senior Windows System Administrator and Troubleshooting Expert.
Problem: My Windows system has logged the following critical event in the past 7 days.

Error Details:
- Severity Level: %s
- Source: %s
- Event ID: %s
- Occurrences (7 Days): %s times

Raw Windows Log Message:
"%s"

Request:
Please analyze this and provide the response in English, structured exactly as follows:
1. Root Cause: Explain simply and non-technically what this error means and why it happened (max 2 lines).
2. Severity: Is this a critical failure requiring immediate action, or safe-to-ignore noise?
3. Step-by-Step Solutions: Provide 3 practical, step-by-step solutions to fix this permanently. Provide exact CMD/PowerShell commands if necessary.`,
		errData.Level, errData.Source, errData.EventID, errData.Count, errData.Message)
}

func fetchTopSystemErrors() {
	atomic.StoreUint32(&isAnalyzerLoading, 1)
	procInvalidateRect.Call(uintptr(hwndAnalyzer), 0, 1)

	// استخراج سطح خطا (LevelDisplayName) به همراه بقیه اطلاعات
	script := `
$ErrorActionPreference = 'SilentlyContinue'
$events = Get-WinEvent -FilterHashtable @{LogName='System'; Level=1,2; StartTime=(Get-Date).AddDays(-7)} -MaxEvents 2000
if ($events) {
	$events | Group-Object Id, ProviderName | Sort-Object Count -Descending | Select-Object -First 5 | ForEach-Object {
		$sample = $_.Group[0]
		$msg = [string]$sample.Message -replace "` + "`" + `r` + "`" + `n", " " -replace "\|", "-"
		$lvl = [string]$sample.LevelDisplayName
		if ([string]::IsNullOrEmpty($lvl)) { $lvl = "Error" }
		Write-Output "$($sample.Id)|$($sample.ProviderName)|$($_.Count)|$msg|$lvl"
	}
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()

	analyzerErrors = []SystemError{}
	if err == nil && len(out) > 0 {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			parts := strings.SplitN(strings.TrimSpace(line), "|", 5)
			if len(parts) == 5 {
				analyzerErrors = append(analyzerErrors, SystemError{
					EventID: parts[0],
					Source:  parts[1],
					Count:   parts[2],
					Message: parts[3],
					Level:   parts[4],
				})
			}
		}
	}

	atomic.StoreUint32(&isAnalyzerLoading, 0)
	procInvalidateRect.Call(uintptr(hwndAnalyzer), 0, 1)

	for i := 0; i < 5; i++ {
		sw := SW_HIDE
		if i < len(analyzerErrors) {
			sw = SW_SHOW
		}
		procShowWindow.Call(uintptr(hwndAskBtns[i]), uintptr(sw))
		procShowWindow.Call(uintptr(hwndCopyBtns[i]), uintptr(sw))
	}
}

func wndProcAnalyzer(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_ERASEBKGND:
		return 1
	case WM_CREATE:
		for i := 0; i < 5; i++ {
			// ارتفاع دکمه‌ها برای هماهنگی با ارتفاع جدید باکس‌ها اصلاح شد (پرش هر ردیف 105 پیکسل)
			hwndCopyBtns[i] = createControl(hwnd, uint32(IDC_COPY_ERR_1+i), "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 510, 110+(i*105), 110, 35)
			hwndAskBtns[i] = createControl(hwnd, uint32(IDC_ASK_AI_1+i), "BUTTON", "", WS_CHILD|BS_OWNERDRAW, 630, 110+(i*105), 130, 35)
		}
		go fetchTopSystemErrors()

	case WM_CTLCOLORSTATIC:
		hdc := syscall.Handle(wParam)
		procSetBkMode.Call(uintptr(hdc), uintptr(TRANSPARENT))
		procSetTextColor.Call(uintptr(hdc), uintptr(RGB(180, 200, 220)))
		return uintptr(hBrushBg)

	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		isAsk := dis.CtlID >= IDC_ASK_AI_1 && dis.CtlID <= IDC_ASK_AI_5
		isCopy := dis.CtlID >= IDC_COPY_ERR_1 && dis.CtlID <= IDC_COPY_ERR_5

		if dis.CtlType == ODT_BUTTON && (isAsk || isCopy) {
			isDown := (dis.ItemState & ODS_SELECTED) != 0
			var borderColor uint32
			text := ""

			if isAsk {
				borderColor = RGB(0, 255, 150)
				text = T("🤖 Ask AI", "🤖 هوش مصنوعی")
			} else {
				borderColor = RGB(255, 180, 0)
				text = T("📋 Copy", "📋 کپی متن")
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

	case WM_COMMAND:
		if int(wParam>>16) == BN_CLICKED {
			cmdID := int(wParam & 0xFFFF)
			if cmdID >= IDC_ASK_AI_1 && cmdID <= IDC_ASK_AI_5 {
				idx := cmdID - IDC_ASK_AI_1
				if idx < len(analyzerErrors) {
					playUIClick()
					prompt := generateAIPrompt(analyzerErrors[idx])
					copyToClipboardNative(prompt)
					openGemini()
				}
			} else if cmdID >= IDC_COPY_ERR_1 && cmdID <= IDC_COPY_ERR_5 {
				idx := cmdID - IDC_COPY_ERR_1
				if idx < len(analyzerErrors) {
					playUIClick()
					prompt := generateAIPrompt(analyzerErrors[idx])
					copyToClipboardNative(prompt)
					ShowMessageBox(hwnd, T("Log data and prompt copied to clipboard!", "اطلاعات لاگ و دستورات با موفقیت کپی شد!"), "Copied", 0x00000040)
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

			procSelectObject.Call(hdc, uintptr(hFontTitle))
			setTextColor(hdc, RGB(0, 210, 255))
			titleRect := RECT{Left: 30, Top: 25, Right: rect.Right, Bottom: 60}
			DrawTextSafe(hdc, T("🧠 Smart Event Log Analyzer", "🧠 تحلیل‌گر هوشمند خطاهای پنهان"), &titleRect, DT_LEFT|DT_TOP)

			if atomic.LoadUint32(&isAnalyzerLoading) == 1 {
				procSelectObject.Call(hdc, uintptr(hFontNormal))
				setTextColor(hdc, RGB(0, 255, 150))
				loadRect := RECT{Left: 30, Top: 100, Right: rect.Right, Bottom: 140}
				DrawTextSafe(hdc, T("⏳ Scanning Windows Event Logs (Level: Critical/Error) - Please wait...", "⏳ در حال اسکن عمیق لاگ‌های ویندوز در ۷ روز گذشته... لطفا صبر کنید"), &loadRect, DT_LEFT|DT_TOP)
			} else {
				if len(analyzerErrors) == 0 {
					procSelectObject.Call(hdc, uintptr(hFontNormal))
					setTextColor(hdc, RGB(0, 255, 150))
					emptyRect := RECT{Left: 30, Top: 100, Right: rect.Right, Bottom: 140}
					DrawTextSafe(hdc, T("✅ System is perfectly healthy! No critical errors found.", "✅ سیستم کاملاً سالم است! هیچ خطای بحرانی در 7 روز گذشته یافت نشد."), &emptyRect, DT_LEFT|DT_TOP)
				} else {
					procSelectObject.Call(hdc, uintptr(hFontWidget))
					yOffset := 80
					rowHeight := 105 // ارتفاع باکس‌ها افزایش یافت

					for _, errItem := range analyzerErrors {
						// باکس پس‌زمینه بزرگ‌تر برای جا دادن متن چند خطی
						drawRect(hdc, 20, yOffset, 780, yOffset+rowHeight-10, 8, RGB(16, 22, 35))

						setTextColor(hdc, RGB(255, 80, 100))
						rTitle := RECT{Left: 35, Top: int32(yOffset + 12), Right: 500, Bottom: int32(yOffset + 30)}
						DrawTextSafe(hdc, fmt.Sprintf("[%s]  Source: %s  |  Event ID: %s  |  Count: %s", errItem.Level, errItem.Source, errItem.EventID, errItem.Count), &rTitle, DT_LEFT|DT_TOP)

						setTextColor(hdc, RGB(180, 200, 220))
						// فضای بیشتر برای پیام
						rDesc := RECT{Left: 35, Top: int32(yOffset + 35), Right: 480, Bottom: int32(yOffset + 85)}

						displayMsg := errItem.Message
						if len(displayMsg) > 200 {
							displayMsg = displayMsg[:197] + "..."
						}

						// استفاده از DT_WORDBREAK (0x00000010) برای رفتن به خط بعد
						if utf16, err := syscall.UTF16FromString(displayMsg); err == nil {
							procDrawText.Call(hdc, uintptr(unsafe.Pointer(&utf16[0])), uintptr(len(utf16)-1), uintptr(unsafe.Pointer(&rDesc)), uintptr(DT_LEFT|DT_TOP|0x00000010))
						}

						yOffset += rowHeight
					}
				}
			}
			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		}

	case WM_CLOSE:
		procShowWindow.Call(uintptr(hwndAnalyzer), SW_HIDE)
		return 0

	default:
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), msg, wParam, lParam)
		return ret
	}
	return 0
}

func OpenAnalyzerWindow() {
	if hwndAnalyzer != 0 {
		procShowWindow.Call(uintptr(hwndAnalyzer), SW_RESTORE)
		user32.NewProc("SetForegroundWindow").Call(uintptr(hwndAnalyzer))
		go fetchTopSystemErrors()
		return
	}

	inst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("SysGuardAnalyzerClass")
	windowName, _ := syscall.UTF16PtrFromString("Smart Diagnostics")

	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(wndProcAnalyzer),
		HInstance:     syscall.Handle(inst),
		HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		LpszClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	// ارتفاع پنجره برای جا دادن ۵ ردیفِ بزرگ‌تر به 680 پیکسل افزایش یافت
	hwndPtr, _, _ := procCreateWindow.Call(0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		uintptr(WS_OVERLAPPEDWINDOW|WS_CLIPCHILDREN), uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT),
		810, 680, uintptr(hwndMain), 0, inst, 0)

	hwndAnalyzer = syscall.Handle(hwndPtr)
	setWindowIcon(hwndAnalyzer, "icon.ico")

	if procDwmSetWindowAttribute.Find() == nil {
		darkMode := int32(1)
		procDwmSetWindowAttribute.Call(uintptr(hwndAnalyzer), 20, uintptr(unsafe.Pointer(&darkMode)), uintptr(unsafe.Sizeof(darkMode)))
	}

	procShowWindow.Call(uintptr(hwndAnalyzer), SW_SHOW)
}
