package main

import (
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

type ParsedHW struct {
	OS, Model, Board, CPU, Cores string // فیلد OS اضافه شد
	RAM, GPU, Disk               []string
	IsLoaded                     bool
}

var (
	sysInfoHwnd syscall.Handle
	currentHW   ParsedHW
)

func FetchDetailedInfo() {
	// اسکریپت آپدیت شد تا ورژن ویندوز را هم بخواند
	script := `$ErrorActionPreference='SilentlyContinue'; $o=Get-CimInstance Win32_OperatingSystem; $s=Get-CimInstance Win32_ComputerSystem; $m=Get-CimInstance Win32_BaseBoard; $c=Get-CimInstance Win32_Processor; Write-Output "OS|$($o.Caption) (Build $($o.Version))"; Write-Output "MODEL|$($s.Manufacturer) $($s.Model)"; Write-Output "BOARD|$($m.Manufacturer) $($m.Product)"; Write-Output "CPU|$($c.Name)"; Write-Output "CORES|$($c.NumberOfCores) Cores / $($c.NumberOfLogicalProcessors) Threads"; foreach($r in @(Get-CimInstance Win32_PhysicalMemory)){$cap=[math]::Round(([double]$r.Capacity)/1GB,2); Write-Output "RAM|$cap GB | $($r.Speed) MHz | $($r.Manufacturer) | P/N: $($r.PartNumber)"}; foreach($g in @(Get-CimInstance Win32_VideoController)){Write-Output "GPU|$($g.Name)"}; foreach($d in @(Get-CimInstance Win32_DiskDrive)){$cap=[math]::Round(([double]$d.Size)/1GB,2); Write-Output "DISK|$($d.Model) ($cap GB)"}`

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, _ := cmd.CombinedOutput()
	lines := strings.Split(string(out), "\n")

	var hw ParsedHW
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			key, val := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			switch key {
			case "OS":
				hw.OS = val
			case "MODEL":
				hw.Model = val
			case "BOARD":
				hw.Board = val
			case "CPU":
				hw.CPU = val
			case "CORES":
				hw.Cores = val
			case "RAM":
				hw.RAM = append(hw.RAM, val)
			case "GPU":
				hw.GPU = append(hw.GPU, val)
			case "DISK":
				hw.Disk = append(hw.Disk, val)
			}
		}
	}
	hw.IsLoaded = true
	currentHW = hw

	if sysInfoHwnd != 0 {
		procInvalidateRect.Call(uintptr(sysInfoHwnd), 0, 1)
	}
}

func wndProcSysInfo(hwnd syscall.Handle, msg uintptr, wParam, lParam uintptr) uintptr {
	switch uint32(msg) {
	case WM_ERASEBKGND:
		return 1
	case 0x0010:
		procShowWindow.Call(uintptr(hwnd), SW_HIDE)
		return 0
	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		if hdc != 0 {
			var rect RECT
			procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&rect)))

			drawRect(hdc, 0, 0, int(rect.Right), int(rect.Bottom), 0, RGB(10, 14, 25))

			if !currentHW.IsLoaded {
				procSelectObject.Call(hdc, uintptr(hFontTitle))
				setTextColor(hdc, RGB(0, 210, 255))
				rMsg := RECT{Left: 0, Top: int32(rect.Bottom/2 - 20), Right: rect.Right, Bottom: rect.Bottom}
				DrawTextSafe(hdc, "⏳ Scanning Deep Hardware Architecture...", &rMsg, DT_CENTER|DT_TOP)
			} else {
				procSelectObject.Call(hdc, uintptr(hFontTitle))
				setTextColor(hdc, RGB(0, 210, 255))
				rTitle := RECT{Left: 30, Top: 25, Right: rect.Right, Bottom: 60}
				DrawTextSafe(hdc, "FULL SYSTEM SPECIFICATIONS", &rTitle, DT_LEFT|DT_TOP)

				procSelectObject.Call(hdc, uintptr(hFontNormal))
				panelCol := RGB(16, 22, 35)

				// پنل 1: سیستم و سیستم‌عامل
				drawRect(hdc, 30, 80, 420, 190, 12, panelCol)
				setTextColor(hdc, RGB(0, 255, 150))
				rBox := RECT{Left: 45, Top: 95, Right: 410, Bottom: 200}
				DrawTextSafe(hdc, "🖥️ System & OS", &rBox, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rBox.Top += 30
				DrawTextSafe(hdc, "OS: "+currentHW.OS, &rBox, DT_LEFT|DT_TOP)
				rBox.Top += 25
				DrawTextSafe(hdc, "Model: "+currentHW.Model, &rBox, DT_LEFT|DT_TOP)
				rBox.Top += 25
				DrawTextSafe(hdc, "Board: "+currentHW.Board, &rBox, DT_LEFT|DT_TOP)

				// پنل 2
				drawRect(hdc, 30, 205, 420, 315, 12, panelCol)
				setTextColor(hdc, RGB(255, 180, 0))
				rBox = RECT{Left: 45, Top: 220, Right: 410, Bottom: 320}
				DrawTextSafe(hdc, "🧠 Processor (CPU)", &rBox, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rBox.Top += 30
				DrawTextSafe(hdc, currentHW.CPU, &rBox, DT_LEFT|DT_TOP)
				rBox.Top += 25
				DrawTextSafe(hdc, currentHW.Cores, &rBox, DT_LEFT|DT_TOP)

				// پنل 3
				drawRect(hdc, 30, 330, 420, 520, 12, panelCol)
				setTextColor(hdc, RGB(255, 80, 100))
				rBox = RECT{Left: 45, Top: 345, Right: 410, Bottom: 520}
				DrawTextSafe(hdc, "🎮 Graphics Adapters (GPU)", &rBox, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rBox.Top += 30
				for _, gpu := range currentHW.GPU {
					DrawTextSafe(hdc, "• "+gpu, &rBox, DT_LEFT|DT_TOP)
					rBox.Top += 25
				}

				// پنل 4
				drawRect(hdc, 440, 80, 850, 290, 12, panelCol)
				setTextColor(hdc, RGB(200, 100, 255))
				rBox = RECT{Left: 455, Top: 95, Right: 840, Bottom: 290}
				DrawTextSafe(hdc, "💾 Memory Modules (RAM Slots)", &rBox, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rBox.Top += 30
				if len(currentHW.RAM) == 0 {
					DrawTextSafe(hdc, "No valid RAM slots detected.", &rBox, DT_LEFT|DT_TOP)
				}
				for i, ram := range currentHW.RAM {
					DrawTextSafe(hdc, fmt.Sprintf("Slot %d: %s", i+1, ram), &rBox, DT_LEFT|DT_TOP)
					rBox.Top += 25
				}

				// پنل 5
				drawRect(hdc, 440, 305, 850, 520, 12, panelCol)
				setTextColor(hdc, RGB(0, 210, 255))
				rBox = RECT{Left: 455, Top: 320, Right: 840, Bottom: 520}
				DrawTextSafe(hdc, "💽 Storage Drives", &rBox, DT_LEFT|DT_TOP)
				setTextColor(hdc, RGB(220, 235, 255))
				rBox.Top += 30
				if len(currentHW.Disk) == 0 {
					DrawTextSafe(hdc, "No external/internal storage found.", &rBox, DT_LEFT|DT_TOP)
				}
				for i, disk := range currentHW.Disk {
					DrawTextSafe(hdc, fmt.Sprintf("Drive %d: %s", i+1, disk), &rBox, DT_LEFT|DT_TOP)
					rBox.Top += 25
				}
			}
			procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&ps)))
		}
	default:
		ret, _, _ := procDefWindowProc.Call(uintptr(hwnd), msg, wParam, lParam)
		return ret
	}
	return 0
}

func InitSysInfoWindow() {
	inst, _, _ := procGetModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("SysGuardInfoClass")
	windowName, _ := syscall.UTF16PtrFromString("Detailed Hardware Specifications")

	wc := WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEX{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(wndProcSysInfo),
		HInstance:     syscall.Handle(inst),
		HbrBackground: syscall.Handle(COLOR_WINDOW + 1),
		LpszClassName: className,
	}
	procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc)))

	hwndPtr, _, _ := procCreateWindow.Call(
		0, uintptr(unsafe.Pointer(className)), uintptr(unsafe.Pointer(windowName)),
		uintptr(WS_OVERLAPPEDWINDOW), uintptr(CW_USEDEFAULT), uintptr(CW_USEDEFAULT),
		890, 580, 0, 0, inst, 0)

	sysInfoHwnd = syscall.Handle(hwndPtr)
	setWindowIcon(sysInfoHwnd, "icon.ico")

	if procDwmSetWindowAttribute.Find() == nil {
		darkMode := int32(1)
		procDwmSetWindowAttribute.Call(uintptr(sysInfoHwnd), 20, uintptr(unsafe.Pointer(&darkMode)), uintptr(unsafe.Sizeof(darkMode)))
	}
}

func OpenSysInfoWindow() {
	if sysInfoHwnd != 0 {
		procShowWindow.Call(uintptr(sysInfoHwnd), SW_SHOW)
		user32.NewProc("SetForegroundWindow").Call(uintptr(sysInfoHwnd))

		if !currentHW.IsLoaded {
			go FetchDetailedInfo()
		}
	}
}
