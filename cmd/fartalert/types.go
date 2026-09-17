package main

import "syscall"

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type PAINTSTRUCT struct {
	Hdc         syscall.Handle
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
}
type WNDCLASSEX struct {
	CbSize, Style               uint32
	LpfnWndProc                 uintptr
	CbClsExtra, CbWndExtra      int32
	HInstance, HIcon, HCursor, HbrBackground syscall.Handle
	LpszMenuName, LpszClassName *uint16
	HIconSm                     syscall.Handle
}
type MSG struct {
	Hwnd           syscall.Handle
	Message        uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             POINT
}
type MEMORYSTATUSEX struct {
	dwLength, dwMemoryLoad                                                                                  uint32
	ullTotalPhys, ullAvailPhys, ullTotalPageFile, ullAvailPageFile, ullTotalVirtual, ullAvailVirtual, ullAvailExtendedVirtual uint64
}
type XINPUT_GAMEPAD struct {
	wButtons                               uint16
	bLeftTrigger, bRightTrigger            byte
	sThumbLX, sThumbLY, sThumbRX, sThumbRY int16
}
type XINPUT_STATE struct {
	dwPacketNumber uint32
	Gamepad        XINPUT_GAMEPAD
}
type SERVICE_STATUS struct {
	dwServiceType, dwCurrentState, dwControlsAccepted, dwWin32ExitCode, dwServiceSpecificExitCode, dwCheckPoint, dwWaitHint uint32
}
type FILETIME struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}
type DRAWITEMSTRUCT struct {
	CtlType    uint32
	CtlID      uint32
	ItemID     uint32
	ItemAction uint32
	ItemState  uint32
	HwndItem   syscall.Handle
	Hdc        syscall.Handle
	RcItem     RECT
	ItemData   uintptr
}