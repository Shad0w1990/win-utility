package main

import (
	"strings"
	"syscall"
	"unsafe"
)

var (
	alertParent   syscall.Handle
	alertMsg      string
	alertTitle    string
	alertColor    uint32
	alertClassReg bool
	hwndAlert     syscall.Handle

	// سایز داینامیک پاپ‌آپ
	alertW int32 = 400
	alertH int32 = 200
)

func alertWndProc(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_CREATE:
		// دکمه همیشه در وسط و پایین قرار می‌گیرد
		btnX := (alertW / 2) - 50
		btnY := alertH - 50
		createControl(hwnd, 1, "BUTTON", "OK", WS_CHILD|WS_VISIBLE|BS_OWNERDRAW, int(btnX), int(btnY), 100, 35)

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

		drawRect(hdc, 0, 0, int(alertW), int(alertH), 0, RGB(10, 14, 25))
		drawRect(hdc, 0, 0, int(alertW), int(alertH), 12, alertColor)
		drawRect(hdc, 2, 2, int(alertW)-4, int(alertH)-4, 10, RGB(16, 22, 35))

		rTitle := RECT{Left: 20, Top: 20, Right: alertW - 20, Bottom: 50}
		procSelectObject.Call(hdc, uintptr(hFontNormal))
		procSetBkMode.Call(hdc, uintptr(TRANSPARENT))
		procSetTextColor.Call(hdc, uintptr(alertColor))
		DrawTextSafe(hdc, alertTitle, &rTitle, DT_CENTER|DT_TOP)

		rMsg := RECT{Left: 30, Top: 60, Right: alertW - 30, Bottom: alertH - 60}
		// استفاده از فونت بزرگتر و خواناتر بجای فونت ریز
		procSelectObject.Call(hdc, uintptr(hFontNormal))
		procSetTextColor.Call(hdc, uintptr(RGB(220, 235, 255)))

		// اگر پیام طولانی است چپ‌چین شود تا خواناتر باشد، اگر کوتاه است وسط‌چین
		format := uint32(DT_CENTER | DT_TOP | 0x00000010)
		if len(alertMsg) > 100 {
			format = uint32(DT_LEFT | DT_TOP | 0x00000010)
		}
		DrawTextSafe(hdc, alertMsg, &rMsg, format)

		procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))

	case WM_DRAWITEM:
		dis := (*DRAWITEMSTRUCT)(unsafe.Pointer(lParam))
		if dis.CtlID == 1 {
			isDown := (dis.ItemState & ODS_SELECTED) != 0
			bg := uint32(RGB(15, 20, 30))
			txt := alertColor

			if isDown {
				bg = alertColor
				txt = RGB(10, 14, 25)
			}
			drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left), int(dis.RcItem.Top), int(dis.RcItem.Right), int(dis.RcItem.Bottom), 8, alertColor)
			drawRect(uintptr(dis.Hdc), int(dis.RcItem.Left)+2, int(dis.RcItem.Top)+2, int(dis.RcItem.Right)-2, int(dis.RcItem.Bottom)-2, 6, bg)

			procSelectObject.Call(uintptr(dis.Hdc), uintptr(hFontWidget))
			procSetBkMode.Call(uintptr(dis.Hdc), uintptr(TRANSPARENT))
			procSetTextColor.Call(uintptr(dis.Hdc), uintptr(txt))
			DrawTextSafe(uintptr(dis.Hdc), "OK", &dis.RcItem, DT_CENTER|DT_VCENTER|DT_SINGLELINE)
			return 1
		}

	case WM_COMMAND:
		if wParam&0xFFFF == 1 {
			procSendMessage.Call(uintptr(hwnd), WM_CLOSE, 0, 0)
		}

	case WM_CLOSE:
		user32.NewProc("EnableWindow").Call(uintptr(alertParent), 1)
		user32.NewProc("DestroyWindow").Call(uintptr(hwnd))
		user32.NewProc("SetForegroundWindow").Call(uintptr(alertParent))
		hwndAlert = 0

	case 0x0084: // WM_NCHITTEST
		return 2

	default:
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), msg, wParam, lParam)
		return ret
	}
	return 0
}

func ShowModernAlert(parent syscall.Handle, message, title string, color uint32) {
	params := &AlertParams{
		Message: message,
		Title:   title,
		Color:   color,
	}
	user32.NewProc("SendMessageW").Call(uintptr(hwndMain), WM_SHOW_ALERT, 0, uintptr(unsafe.Pointer(params)))
}

func RenderModernAlert(parent syscall.Handle, message, title string, color uint32) {
	if hwndAlert != 0 {
		return
	}
	alertParent = parent
	alertMsg = message
	alertTitle = title
	alertColor = color

	// محاسبه‌ی بسیار دقیق سایز با توجه به طول متن
	runeCount := int32(len([]rune(message)))
	lineCount := int32(strings.Count(message, "\n"))
	estimatedLines := runeCount / 45
	if estimatedLines > lineCount {
		lineCount = estimatedLines
	}

	alertW = 400
	alertH = 200

	if lineCount > 2 || runeCount > 60 {
		alertW = 600
		alertH = 160 + (lineCount * 30) // افزایش ارتفاع بر اساس تعداد خطوط و فونت بزرگتر
	}
	if alertH > 650 {
		alertH = 650
	}

	if !alertClassReg {
		className, _ := syscall.UTF16PtrFromString("SysGuardAlertClass")
		wc := WNDCLASSEX{
			CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
			LpfnWndProc:   syscall.NewCallback(alertWndProc),
			HCursor:       syscall.Handle(0),
			HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
			LpszClassName: className,
		}
		procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
		alertClassReg = true
	}

	user32.NewProc("EnableWindow").Call(uintptr(parent), 0)

	className, _ := syscall.UTF16PtrFromString("SysGuardAlertClass")
	hwndPtr, _, _ := procCreateWindow.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		0,
		uintptr(WS_POPUP|WS_VISIBLE),
		0, 0, uintptr(alertW), uintptr(alertH),
		uintptr(parent), 0, 0, 0)

	hwndAlert = syscall.Handle(hwndPtr)

	var rect RECT
	user32.NewProc("GetWindowRect").Call(uintptr(parent), uintptr(unsafe.Pointer(&rect)))
	cx := int(rect.Left) + (int(rect.Right-rect.Left) / 2) - int(alertW/2)
	cy := int(rect.Top) + (int(rect.Bottom-rect.Top) / 2) - int(alertH/2)
	user32.NewProc("SetWindowPos").Call(uintptr(hwndAlert), 0, uintptr(cx), uintptr(cy), 0, 0, 0x0001|0x0004)

	procShowWindow.Call(uintptr(hwndAlert), SW_SHOW)
	procUpdateWindow.Call(uintptr(hwndAlert))
}
