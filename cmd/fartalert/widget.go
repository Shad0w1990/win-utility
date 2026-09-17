package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

func wndProcWidget(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_ERASEBKGND:
		return 1
	case WM_LBUTTONDOWN:
		user32.NewProc("ReleaseCapture").Call()
		procSendMessage.Call(uintptr(hwnd), uintptr(WM_NCLBUTTONDOWN), uintptr(HTCAPTION), 0)
		return 0
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			var rect RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

			rgbColor := HSVtoRGB(globalHue, 1.0, 1.0)

			drawRect(hdc, 0, 0, int(rect.Right), int(rect.Bottom), 0, RGB(0, 0, 0))
			drawRect(hdc, 0, 0, int(rect.Right), int(rect.Bottom), 15, rgbColor)
			drawRect(hdc, 2, 2, int(rect.Right)-2, int(rect.Bottom)-2, 12, RGB(12, 16, 25))

			procSelectObject.Call(hdc, uintptr(hFontWidget))

			setTextColor(hdc, rgbColor)
			rect.Top = 10
			rect.Left = 15
			DrawTextSafe(hdc, "SysGuard Overlay (Drag Me)", &rect, DT_LEFT|DT_TOP)

			setTextColor(hdc, RGB(255, 255, 255))
			rect.Top = 40
			DrawTextSafe(hdc, fmt.Sprintf("CPU: %.1f%%  |  RAM: %.0f%%", cpuUsageVal, ramUsageVal), &rect, DT_LEFT|DT_TOP)

			setTextColor(hdc, RGB(255, 100, 100))
			rect.Top = 65
			DrawTextSafe(hdc, fmt.Sprintf("GPU: %.0f%%  |  Temp: %.0f°C", gpuLoadVal, gpuTempVal), &rect, DT_LEFT|DT_TOP)

			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		}
	default:
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), msg, wParam, lParam)
		return ret
	}
	return 0
}

func createWidget() {
	inst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("SysGuardWidgetClass")
	wc := WNDCLASSEX{
		CbSize: uint32(unsafe.Sizeof(WNDCLASSEX{})), Style: CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc: syscall.NewCallback(wndProcWidget), HInstance: syscall.Handle(inst),
		HbrBackground: syscall.Handle(COLOR_WINDOW), LpszClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))
	hwndPtr, _, _ := procCreateWindow.Call(uintptr(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW), uintptr(unsafe.Pointer(className)), 0,
		uintptr(WS_POPUP), 20, 20, 360, 110, 0, 0, inst, 0)
	hwndWidget = syscall.Handle(hwndPtr)

	procSetLayeredWindowAttr.Call(uintptr(hwndWidget), uintptr(RGB(0, 0, 0)), 210, uintptr(LWA_COLORKEY|LWA_ALPHA))
}
