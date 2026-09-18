package main

import (
	"syscall"
)

// DLLها و پروکجرهای Win32 API
var (
	user32   = syscall.NewLazyDLL("user32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	xinput   = syscall.NewLazyDLL("xinput1_4.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	powrprof = syscall.NewLazyDLL("powrprof.dll")
	ole32    = syscall.NewLazyDLL("ole32.dll")

	procGetMessage            = user32.NewProc("GetMessageW")
	procTranslateMsg          = user32.NewProc("TranslateMessage")
	procDispatchMsg           = user32.NewProc("DispatchMessageW")
	procCreateWindow          = user32.NewProc("CreateWindowExW")
	procDefWindowProc         = user32.NewProc("DefWindowProcW")
	procRegisterClassEx       = user32.NewProc("RegisterClassExW")
	procLoadCursor            = user32.NewProc("LoadCursorW")
	procLoadImage             = user32.NewProc("LoadImageW")
	procBeginPaint            = user32.NewProc("BeginPaint")
	procEndPaint              = user32.NewProc("EndPaint")
	procGetClientRect         = user32.NewProc("GetClientRect")
	procDrawText              = user32.NewProc("DrawTextW")
	procInvalidateRect        = user32.NewProc("InvalidateRect")
	procUpdateWindow          = user32.NewProc("UpdateWindow")
	procMessageBox            = user32.NewProc("MessageBoxW")
	procShowWindow            = user32.NewProc("ShowWindow")
	procSetTimer              = user32.NewProc("SetTimer")
	procSendMessage           = user32.NewProc("SendMessageW")
	procSetLayeredWindowAttr  = user32.NewProc("SetLayeredWindowAttributes")
	procDwmSetWindowAttribute = dwmapi.NewProc("DwmSetWindowAttribute")
	procKeybdEvent            = user32.NewProc("keybd_event")
	procSetWindowText         = user32.NewProc("SetWindowTextW")

	// توابع بومی ویندوز برای مدیریت کلیپ‌بورد (اضافه شده)
	procOpenClipboard    = user32.NewProc("OpenClipboard")
	procCloseClipboard   = user32.NewProc("CloseClipboard")
	procEmptyClipboard   = user32.NewProc("EmptyClipboard")
	procSetClipboardData = user32.NewProc("SetClipboardData")

	procGetModuleHandle           = kernel32.NewProc("GetModuleHandleW")
	procGetSystemPowerStatus      = kernel32.NewProc("GetSystemPowerStatus")
	procBeep                      = kernel32.NewProc("Beep")
	procGlobalMemoryStatus        = kernel32.NewProc("GlobalMemoryStatusEx")
	procGetDiskFreeSpace          = kernel32.NewProc("GetDiskFreeSpaceExW")
	procGetSystemTimes            = kernel32.NewProc("GetSystemTimes")
	procOpenProcess               = kernel32.NewProc("OpenProcess")
	procCloseHandle               = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageName = kernel32.NewProc("QueryFullProcessImageNameW")
	procLocalFree                 = kernel32.NewProc("LocalFree")
	procGetTickCount64            = kernel32.NewProc("GetTickCount64")

	// توابع مدیریت حافظه برای کپی کردن متون طولانی در کلیپ‌بورد (اضافه شده)
	procGlobalAlloc  = kernel32.NewProc("GlobalAlloc")
	procGlobalLock   = kernel32.NewProc("GlobalLock")
	procGlobalUnlock = kernel32.NewProc("GlobalUnlock")

	procXInputGetState     = xinput.NewProc("XInputGetState")
	procOpenSCManager      = advapi32.NewProc("OpenSCManagerW")
	procOpenService        = advapi32.NewProc("OpenServiceW")
	procQueryServiceStatus = advapi32.NewProc("QueryServiceStatus")
	procCloseServiceHandle = advapi32.NewProc("CloseServiceHandle")

	procPowerGetActiveScheme  = powrprof.NewProc("PowerGetActiveScheme")
	procPowerReadFriendlyName = powrprof.NewProc("PowerReadFriendlyName")

	procCoInitialize     = ole32.NewProc("CoInitialize")
	procCoUninitialize   = ole32.NewProc("CoUninitialize")
	procCoCreateInstance = ole32.NewProc("CoCreateInstance")

	procCreateSolidBrush = gdi32.NewProc("CreateSolidBrush")
	procSelectObject     = gdi32.NewProc("SelectObject")
	procDeleteObject     = gdi32.NewProc("DeleteObject")
	procEllipse          = gdi32.NewProc("Ellipse")
	procRoundRect        = gdi32.NewProc("RoundRect")
	procRectangle        = gdi32.NewProc("Rectangle")
	procSetTextColor     = gdi32.NewProc("SetTextColor")
	procSetBkMode        = gdi32.NewProc("SetBkMode")
	procCreateFontW      = gdi32.NewProc("CreateFontW")

	procCreatePen = gdi32.NewProc("CreatePen")
	procMoveToEx  = gdi32.NewProc("MoveToEx")
	procLineTo    = gdi32.NewProc("LineTo")
	procPolygon   = gdi32.NewProc("Polygon")

	winmm       = syscall.NewLazyDLL("winmm.dll")
	procMciSend = winmm.NewProc("mciSendStringW")
)

// شناسه‌های اختصاصی هسته ویندوز برای صدا
var (
	CLSID_MMDeviceEnumerator = syscall.GUID{Data1: 0xBCDE0395, Data2: 0xE52F, Data3: 0x467C, Data4: [8]byte{0x8E, 0x3D, 0xC4, 0x57, 0x92, 0x91, 0x69, 0x2E}}
	IID_IMMDeviceEnumerator  = syscall.GUID{Data1: 0xA95664D2, Data2: 0x9614, Data3: 0x4F35, Data4: [8]byte{0xA7, 0x46, 0xDE, 0x8D, 0xB6, 0x36, 0x17, 0xE6}}
	IID_IAudioEndpointVolume = syscall.GUID{Data1: 0x5CDF2C82, Data2: 0x841E, Data3: 0x4546, Data4: [8]byte{0x97, 0x22, 0x0C, 0xF7, 0x40, 0x78, 0x22, 0x9A}}
)
