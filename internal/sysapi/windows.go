package sysapi

import (
	"syscall"
	"unsafe"
)

var (
	modKernel32 = syscall.NewLazyDLL("kernel32.dll")
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modPsapi    = syscall.NewLazyDLL("psapi.dll")

	procCreateToolhelp32Snapshot = modKernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32First           = modKernel32.NewProc("Process32FirstW")
	procProcess32Next            = modKernel32.NewProc("Process32NextW")
	procCloseHandle              = modKernel32.NewProc("CloseHandle")
	procGetProcessMemoryInfo     = modPsapi.NewProc("GetProcessMemoryInfo")
)

const (
	TH32CS_SNAPPROCESS = 0x00000002
)

type ProcessEntry32 struct {
	Size              uint32
	Usage             uint32
	ProcessID         uint32
	DefaultHeapID     uintptr
	ModuleID          uint32
	Threads           uint32
	ParentProcessID   uint32
	PriClassBase      int32
	Flags             uint32
	ExeFile           [260]uint16
}

type ProcessMemoryCountersEx struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

func GetProcesses() ([]uint32, []string, error) {
	snapshot, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snapshot == uintptr(syscall.InvalidHandle) {
		return nil, nil, syscall.GetLastError()
	}
	defer procCloseHandle.Call(snapshot)

	var entry ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	ret, _, _ := procProcess32First.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
	if ret == 0 {
		return nil, nil, syscall.GetLastError()
	}

	var pids []uint32
	var names []string

	for {
		pids = append(pids, entry.ProcessID)
		names = append(names, syscall.UTF16ToString(entry.ExeFile[:]))

		ret, _, _ = procProcess32Next.Call(snapshot, uintptr(unsafe.Pointer(&entry)))
		if ret == 0 {
			break
		}
	}

	return pids, names, nil
}
