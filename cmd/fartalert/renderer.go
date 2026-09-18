package main

import (
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

//go:embed icon.ico
var iconBytes []byte

func RGB(r, g, b byte) uint32 { return uint32(r) | (uint32(g) << 8) | (uint32(b) << 16) }

func HSVtoRGB(h, s, v float64) uint32 {
	c := v * s
	x := c * (1 - math.Abs(math.Mod(h/60.0, 2)-1))
	m := v - c
	var r, g, b float64
	switch {
	case h >= 0 && h < 60:
		r, g, b = c, x, 0
	case h >= 60 && h < 120:
		r, g, b = x, c, 0
	case h >= 120 && h < 180:
		r, g, b = 0, c, x
	case h >= 180 && h < 240:
		r, g, b = 0, x, c
	case h >= 240 && h < 300:
		r, g, b = x, 0, c
	case h >= 300 && h < 360:
		r, g, b = c, 0, x
	}
	return RGB(byte((r+m)*255), byte((g+m)*255), byte((b+m)*255))
}

func setWindowIcon(hwnd syscall.Handle, iconName string) {
	tempPath := filepath.Join(os.TempDir(), "sysguard_temp_icon.ico")
	os.WriteFile(tempPath, iconBytes, 0644)
	pathPtr, err := syscall.UTF16PtrFromString(tempPath)
	if err != nil {
		return
	}
	hIcon, _, _ := procLoadImage.Call(0, uintptr(unsafe.Pointer(pathPtr)), uintptr(IMAGE_ICON), 0, 0, uintptr(LR_LOADFROMFILE|LR_DEFAULTSIZE))
	if hIcon != 0 {
		procSendMessage.Call(uintptr(hwnd), uintptr(WM_SETICON), 0, hIcon)
		procSendMessage.Call(uintptr(hwnd), uintptr(WM_SETICON), 1, hIcon)
	}
}

func createModernFont(size, weight int) syscall.Handle {
	fontName, _ := syscall.UTF16PtrFromString("Segoe UI")
	ret, _, _ := procCreateFontW.Call(uintptr(size), 0, 0, 0, uintptr(weight), 0, 0, 0, 1, 0, 0, uintptr(CLEARTYPE_QUALITY), 0, uintptr(unsafe.Pointer(fontName)))
	return syscall.Handle(ret)
}

func drawRect(hdc uintptr, left, top, right, bottom int, radius int, color uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	if radius > 0 {
		procRoundRect.Call(hdc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom), uintptr(radius), uintptr(radius))
	} else {
		procRectangle.Call(hdc, uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
	}
	procSelectObject.Call(hdc, oldBrush)
	procDeleteObject.Call(brush)
}

func drawCircle(hdc uintptr, x, y, radius int, color uint32) {
	brush, _, _ := procCreateSolidBrush.Call(uintptr(color))
	oldBrush, _, _ := procSelectObject.Call(hdc, brush)
	procEllipse.Call(hdc, uintptr(x-radius), uintptr(y-radius), uintptr(x+radius), uintptr(y+radius))
	procSelectObject.Call(hdc, oldBrush)
	procDeleteObject.Call(brush)
}

func setTextColor(hdc uintptr, color uint32) {
	procSetBkMode.Call(hdc, uintptr(TRANSPARENT))
	procSetTextColor.Call(hdc, uintptr(color))
}

func drawGridGraph(hdc uintptr, x, y, w, h int, history []float64, maxVal float64, lineColor, fillColor uint32) {
	drawRect(hdc, x, y, x+w, y+h, 0, RGB(12, 16, 24))

	hPenGrid, _, _ := procCreatePen.Call(0, 1, uintptr(RGB(25, 35, 50)))
	oldPen, _, _ := procSelectObject.Call(hdc, hPenGrid)
	for i := 1; i < 10; i++ {
		gx := x + (w * i / 10)
		procMoveToEx.Call(hdc, uintptr(gx), uintptr(y), 0)
		procLineTo.Call(hdc, uintptr(gx), uintptr(y+h))
		gy := y + (h * i / 10)
		procMoveToEx.Call(hdc, uintptr(x), uintptr(gy), 0)
		procLineTo.Call(hdc, uintptr(x+w), uintptr(gy))
	}
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(hPenGrid)

	if maxVal <= 0 {
		maxVal = 1
	}

	pts := make([]POINT, len(history)+2)
	stepX := float64(w) / float64(len(history)-1)
	for i, val := range history {
		px := x + int(float64(i)*stepX)
		py := y + h - int((val/maxVal)*float64(h))
		if py < y {
			py = y
		}
		if py > y+h {
			py = y + h
		}
		pts[i] = POINT{X: int32(px), Y: int32(py)}
	}
	pts[len(history)] = POINT{X: int32(x + w), Y: int32(y + h)}
	pts[len(history)+1] = POINT{X: int32(x), Y: int32(y + h)}

	hBrushFill, _, _ := procCreateSolidBrush.Call(uintptr(fillColor))
	hPenNull, _, _ := procCreatePen.Call(PS_NULL, 0, 0)
	procSelectObject.Call(hdc, hBrushFill)
	procSelectObject.Call(hdc, hPenNull)
	procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	procDeleteObject.Call(hBrushFill)
	procDeleteObject.Call(hPenNull)

	hPenLine, _, _ := procCreatePen.Call(0, 2, uintptr(lineColor))
	procSelectObject.Call(hdc, hPenLine)
	for i := 0; i < len(history); i++ {
		if i == 0 {
			procMoveToEx.Call(hdc, uintptr(pts[i].X), uintptr(pts[i].Y), 0)
		} else {
			procLineTo.Call(hdc, uintptr(pts[i].X), uintptr(pts[i].Y))
		}
	}
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(hPenLine)
}

func drawAdvancedNetworkGraph(hdc uintptr, x, y, w, h int, dlHist, ulHist []float64, maxVal float64) {
	drawRect(hdc, x, y, x+w, y+h, 0, RGB(12, 16, 24))

	hPenGrid, _, _ := procCreatePen.Call(0, 1, uintptr(RGB(25, 40, 60)))
	oldPen, _, _ := procSelectObject.Call(hdc, hPenGrid)
	for i := 1; i < 10; i++ {
		gx := x + (w * i / 10)
		procMoveToEx.Call(hdc, uintptr(gx), uintptr(y), 0)
		procLineTo.Call(hdc, uintptr(gx), uintptr(y+h))
	}
	for i := 1; i < 5; i++ {
		gy := y + (h * i / 5)
		procMoveToEx.Call(hdc, uintptr(x), uintptr(gy), 0)
		procLineTo.Call(hdc, uintptr(x+w), uintptr(gy))
	}
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(hPenGrid)

	if maxVal <= 0 {
		maxVal = 1
	}
	stepX := float64(w) / float64(len(dlHist)-1)

	pts := make([]POINT, len(dlHist)+2)
	for i, val := range dlHist {
		px := x + int(float64(i)*stepX)
		py := y + h - int((val/maxVal)*float64(h))
		if py < y {
			py = y
		}
		pts[i] = POINT{X: int32(px), Y: int32(py)}
	}
	pts[len(dlHist)] = POINT{X: int32(x + w), Y: int32(y + h)}
	pts[len(dlHist)+1] = POINT{X: int32(x), Y: int32(y + h)}

	hBrushFill, _, _ := procCreateSolidBrush.Call(uintptr(RGB(10, 40, 60)))
	hPenNull, _, _ := procCreatePen.Call(PS_NULL, 0, 0)
	procSelectObject.Call(hdc, hBrushFill)
	procSelectObject.Call(hdc, hPenNull)
	procPolygon.Call(hdc, uintptr(unsafe.Pointer(&pts[0])), uintptr(len(pts)))
	procDeleteObject.Call(hBrushFill)
	procDeleteObject.Call(hPenNull)

	drawDataLine := func(data []float64, color uint32) {
		hPen, _, _ := procCreatePen.Call(0, 2, uintptr(color))
		procSelectObject.Call(hdc, hPen)
		for i, val := range data {
			px := x + int(float64(i)*stepX)
			py := y + h - int((val/maxVal)*float64(h))
			if py < y {
				py = y
			}
			if py > y+h {
				py = y + h
			}
			if i == 0 {
				procMoveToEx.Call(hdc, uintptr(px), uintptr(py), 0)
			} else {
				procLineTo.Call(hdc, uintptr(px), uintptr(py))
			}
		}
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
	}

	drawDataLine(dlHist, RGB(0, 210, 255))
	drawDataLine(ulHist, RGB(255, 80, 200))

	maxLabel := fmt.Sprintf("%.1f MB/s", (maxVal/1024.0)/1024.0)
	if maxVal < 1024*1024 {
		maxLabel = fmt.Sprintf("%.0f KB/s", maxVal/1024.0)
	}

	midLabel := fmt.Sprintf("%.1f", (maxVal/1024.0)/2) + " KB/s"
	if maxVal >= 1024*1024 {
		midLabel = fmt.Sprintf("%.1f", (maxVal/1024.0/1024.0)/2) + " MB/s"
	}

	setTextColor(hdc, RGB(150, 170, 190))
	rTop := RECT{Left: int32(x + 5), Top: int32(y + 2), Right: int32(x + w), Bottom: int32(y + 20)}
	DrawTextSafe(hdc, maxLabel, &rTop, DT_LEFT|DT_TOP)

	rMid := RECT{Left: int32(x + 5), Top: int32(y + h/2 - 8), Right: int32(x + w), Bottom: int32(y + h)}
	DrawTextSafe(hdc, midLabel, &rMid, DT_LEFT|DT_TOP)
}

func drawTempGauge(hdc uintptr, cx, cy, radius int, temp float64) {
	for angle := 180; angle >= 0; angle -= 6 {
		rad := float64(angle) * math.Pi / 180.0
		x1 := cx + int(float64(radius-6)*math.Cos(rad))
		y1 := cy - int(float64(radius-6)*math.Sin(rad))
		x2 := cx + int(float64(radius)*math.Cos(rad))
		y2 := cy - int(float64(radius)*math.Sin(rad))

		ratio := float64(180-angle) / 180.0
		var c uint32
		if ratio < 0.5 {
			c = RGB(0, 210, 150)
		} else if ratio < 0.75 {
			c = RGB(255, 180, 0)
		} else {
			c = RGB(255, 50, 50)
		}

		hPen, _, _ := procCreatePen.Call(0, 3, uintptr(c))
		oldPen, _, _ := procSelectObject.Call(hdc, hPen)
		procMoveToEx.Call(hdc, uintptr(x1), uintptr(y1), 0)
		procLineTo.Call(hdc, uintptr(x2), uintptr(y2))
		procSelectObject.Call(hdc, oldPen)
		procDeleteObject.Call(hPen)
	}

	valAngle := 180.0 - (temp/150.0)*180.0
	if valAngle < 0 {
		valAngle = 0
	}
	rad := valAngle * math.Pi / 180.0
	nx := cx + int(float64(radius-10)*math.Cos(rad))
	ny := cy - int(float64(radius-10)*math.Sin(rad))

	hPenNeedle, _, _ := procCreatePen.Call(0, 2, uintptr(RGB(255, 255, 255)))
	oldPen, _, _ := procSelectObject.Call(hdc, hPenNeedle)
	procMoveToEx.Call(hdc, uintptr(cx), uintptr(cy), 0)
	procLineTo.Call(hdc, uintptr(nx), uintptr(ny))
	procSelectObject.Call(hdc, oldPen)
	procDeleteObject.Call(hPenNeedle)

	drawCircle(hdc, cx, cy, 4, RGB(255, 255, 255))

	rect := RECT{Left: int32(cx - 30), Top: int32(cy + 4), Right: int32(cx + 30), Bottom: int32(cy + 25)}
	setTextColor(hdc, RGB(200, 220, 240))
	DrawTextSafe(hdc, fmt.Sprintf("%.0f°C", temp), &rect, DT_CENTER|DT_TOP)
}
