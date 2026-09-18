package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"
)

func getWmiInfo(query string) string {
	cmd := exec.Command("powershell", "-NoProfile", "-Command", query)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func fileTimeToUint64(ft FILETIME) uint64 {
	return uint64(ft.DwHighDateTime)<<32 | uint64(ft.DwLowDateTime)
}

func getCPUUsage() float64 {
	var idleTime, kernelTime, userTime FILETIME
	ret, _, _ := procGetSystemTimes.Call(uintptr(unsafe.Pointer(&idleTime)), uintptr(unsafe.Pointer(&kernelTime)), uintptr(unsafe.Pointer(&userTime)))
	if ret == 0 {
		return 0.0
	}
	idle, kernel, user := fileTimeToUint64(idleTime), fileTimeToUint64(kernelTime), fileTimeToUint64(userTime)
	sysDelta := (kernel - prevKernel) + (user - prevUser)
	idleDelta := idle - prevIdle
	prevIdle, prevKernel, prevUser = idle, kernel, user
	if sysDelta == 0 {
		return 0.0
	}
	return float64(sysDelta-idleDelta) * 100.0 / float64(sysDelta)
}

func fetchStaticHardwareInfo() {
	cpuName := getWmiInfo("(Get-CimInstance Win32_Processor).Name")
	if cpuName != "" {
		for strings.Contains(cpuName, "  ") {
			cpuName = strings.ReplaceAll(cpuName, "  ", " ")
		}
		sysCpuName = cpuName
	}

	sysModel := getWmiInfo("(Get-CimInstance Win32_ComputerSystem).Model")
	sysMaker := getWmiInfo("(Get-CimInstance Win32_ComputerSystem).Manufacturer")

	if sysModel == "System Product Name" || sysModel == "To be filled by O.E.M." || sysModel == "" {
		boardMaker := getWmiInfo("(Get-CimInstance Win32_BaseBoard).Manufacturer")
		boardModel := getWmiInfo("(Get-CimInstance Win32_BaseBoard).Product")
		sysModelName = boardMaker + " " + boardModel
	} else {
		sysModelName = sysMaker + " " + sysModel
	}

	sysModelName = strings.ReplaceAll(sysModelName, "ASUSTeK COMPUTER INC.", "ASUS")
	isAsus = strings.Contains(strings.ToUpper(sysModelName), "ASUS")

	var status struct {
		ACLineStatus, BatteryFlag, BatteryLifePercent, Reserved1 byte
		BatteryLifeTime, BatteryFullLifeTime                     uint32
	}
	ret, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	isLaptop = (ret != 0 && status.BatteryFlag != 128)

	disk := getWmiInfo("(Get-PhysicalDisk | Select-Object -First 1).MediaType")
	if disk != "" {
		diskMediaType = disk
	}
}

func getGPUStatsExt() (string, string, float64, float64) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=name,power.draw,utilization.gpu,temperature.gpu", "--format=csv,noheader,nounits")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "NVIDIA GeForce GPU", "120 W", 0, 42
	}
	parts := strings.Split(strings.TrimSpace(string(out)), ",")
	if len(parts) >= 4 {
		var util, temp float64
		fmt.Sscanf(strings.TrimSpace(parts[2]), "%f", &util)
		fmt.Sscanf(strings.TrimSpace(parts[3]), "%f", &temp)
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]) + " W", util, temp
	}
	return "NVIDIA GeForce GPU", "120 W", 0, 42
}

func startTelemetryLogger() {
	appData := os.Getenv("LOCALAPPDATA")
	dir := filepath.Join(appData, "SysGuard")
	os.MkdirAll(dir, 0755)
	telemetryFile := filepath.Join(dir, "telemetry_1month.json")

	data, err := os.ReadFile(telemetryFile)
	if err == nil {
		json.Unmarshal(data, &telemetryHistory)
	}

	thirtyDaysAgo := time.Now().Unix() - (30 * 24 * 60 * 60)
	var filtered []TelemetryData
	for _, t := range telemetryHistory {
		if t.Timestamp > thirtyDaysAgo {
			filtered = append(filtered, t)
		}
	}
	telemetryHistory = filtered

	go func() {
		for {
			time.Sleep(10 * time.Minute)
			cSum, gSum := 0.0, 0.0
			for _, v := range cpuHistory {
				cSum += v
			}
			for _, v := range gpuHistory {
				gSum += v
			}
			telemetryHistory = append(telemetryHistory, TelemetryData{
				Timestamp: time.Now().Unix(),
				CPU:       cSum / float64(len(cpuHistory)),
				RAM:       ramUsageVal,
				GPU:       gSum / float64(len(gpuHistory)),
				Disk:      diskUsageVal,
			})
			b, _ := json.Marshal(telemetryHistory)
			os.WriteFile(telemetryFile, b, 0644)
		}
	}()
}

func startHardwareScanner() {
	startTelemetryLogger()
	go func() {
		fetchStaticHardwareInfo()
		getCPUUsage()
		for {
			cpuUsageVal = getCPUUsage()
			for i := 0; i < len(cpuHistory)-1; i++ {
				cpuHistory[i] = cpuHistory[i+1]
			}
			cpuHistory[len(cpuHistory)-1] = cpuUsageVal

			var mem MEMORYSTATUSEX
			mem.dwLength = uint32(unsafe.Sizeof(mem))
			procGlobalMemoryStatus.Call(uintptr(unsafe.Pointer(&mem)))
			totalRamGB = float64(mem.ullTotalPhys) / (1024 * 1024 * 1024)
			ramUsageVal = float64(mem.dwMemoryLoad)
			for i := 0; i < len(ramHistory)-1; i++ {
				ramHistory[i] = ramHistory[i+1]
			}
			ramHistory[len(ramHistory)-1] = ramUsageVal

			n, p, u, t := getGPUStatsExt()
			sysGpuName = n
			sysGpuPower = p
			gpuLoadVal = u
			gpuTempVal = t
			for i := 0; i < len(gpuHistory)-1; i++ {
				gpuHistory[i] = gpuHistory[i+1]
			}
			gpuHistory[len(gpuHistory)-1] = gpuLoadVal

			var free, total, freeTotal uint64
			cStr, _ := syscall.UTF16PtrFromString("C:\\")
			procGetDiskFreeSpace.Call(uintptr(unsafe.Pointer(cStr)), uintptr(unsafe.Pointer(&free)), uintptr(unsafe.Pointer(&total)), uintptr(unsafe.Pointer(&freeTotal)))
			if total > 0 {
				diskTotalGB = float64(total) / (1024 * 1024 * 1024)
				diskFreeGB = float64(free) / (1024 * 1024 * 1024)
				diskUsageVal = float64(total-free) * 100.0 / float64(total)
			}
			for i := 0; i < len(diskHistory)-1; i++ {
				diskHistory[i] = diskHistory[i+1]
			}
			diskHistory[len(diskHistory)-1] = diskUsageVal

			if currentPage == 0 {
				procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
			}
			time.Sleep(1 * time.Second)
		}
	}()

	go func() {
		wasConnected := false
		for {
			var state XINPUT_STATE
			ret, _, _ := procXInputGetState.Call(0, uintptr(unsafe.Pointer(&state)))

			if ret == 0 {
				if !wasConnected {
					query := `(Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.FriendlyName -match 'Xbox|Controller|Gamepad|DualSense|DualShock' -and $_.Class -ne 'AudioEndpoint' -and $_.Class -ne 'Audio' } | Select-Object -First 1).FriendlyName`
					name := getWmiInfo(query)
					if name != "" {
						sysGamepadName = name
					} else {
						sysGamepadName = "Standard XInput Controller"
					}
					wasConnected = true
				}
			} else {
				sysGamepadName = "Searching..."
				wasConnected = false
			}
			time.Sleep(3 * time.Second)
		}
	}()
}

func startGraphScanner() {
	go func() {
		var lastR, lastS uint64
		for {
			cmd := exec.Command("cmd", "/c", "netstat -e")
			cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmd.Output()
			if err == nil {
				fields := strings.Fields(string(out))
				var r, s uint64
				for i, f := range fields {
					if f == "Bytes" && i+2 < len(fields) {
						fmt.Sscanf(fields[i+1], "%d", &r)
						fmt.Sscanf(fields[i+2], "%d", &s)
						break
					}
				}
				if lastR != 0 || lastS != 0 {
					deltaR := float64(r - lastR)
					deltaS := float64(s - lastS)

					for i := 0; i < len(netDLHistory)-1; i++ {
						netDLHistory[i] = netDLHistory[i+1]
						netULHistory[i] = netULHistory[i+1]
					}
					netDLHistory[len(netDLHistory)-1] = deltaR
					netULHistory[len(netULHistory)-1] = deltaS

					max := 1.0
					for i := range netDLHistory {
						if netDLHistory[i] > max {
							max = netDLHistory[i]
						}
						if netULHistory[i] > max {
							max = netULHistory[i]
						}
					}
					maxNetSpeed = max

					dlKbps := deltaR / 1024.0
					ulKbps := deltaS / 1024.0

					if dlKbps > 1024 {
						currentDLText = fmt.Sprintf("DL: %.1f MB/s", dlKbps/1024.0)
					} else {
						currentDLText = fmt.Sprintf("DL: %.1f KB/s", dlKbps)
					}

					if ulKbps > 1024 {
						currentULText = fmt.Sprintf("UL: %.1f MB/s", ulKbps/1024.0)
					} else {
						currentULText = fmt.Sprintf("UL: %.1f KB/s", ulKbps)
					}

					if currentPage == 1 {
						procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
					}
				}
				lastR, lastS = r, s
			}
			time.Sleep(1 * time.Second)
		}
	}()
}

// فیلتر کردن هوشمند آی‌پی‌ها برای دور زدن آداپتورهای مجازی و قطع شده
func fetchLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Offline"
	}
	var fallbackIP string
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				ipStr := ip4.String()
				// رد کردن آی‌پی‌های محلی و بدون اینترنت
				if strings.HasPrefix(ipStr, "169.254.") {
					fallbackIP = ipStr
					continue
				}
				// اولویت پایین برای آداپتورهای مجازی معروف مثل VBox و VMWare
				if strings.HasPrefix(ipStr, "192.168.56.") || strings.HasPrefix(ipStr, "192.168.137.") {
					if fallbackIP == "" {
						fallbackIP = ipStr
					}
					continue
				}
				return ipStr
			}
		}
	}
	if fallbackIP != "" {
		return fallbackIP
	}
	return "Offline"
}

func getProcessNameNative(pid string) string {
	pidInt, _ := strconv.Atoi(pid)
	hProcess, _, _ := procOpenProcess.Call(0x1000, 0, uintptr(pidInt))
	if hProcess == 0 {
		return "Unknown"
	}
	defer procCloseHandle.Call(hProcess)

	buf := make([]uint16, 260)
	size := uint32(260)
	ret, _, _ := procQueryFullProcessImageName.Call(hProcess, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ret != 0 {
		fullPath := syscall.UTF16ToString(buf)
		return filepath.Base(fullPath)
	}
	return "Unknown"
}

func startNetworkScanner() {
	go func() {
		for {
			sysIPAddress = "IP: " + fetchLocalIP()

			netUpdateText = "Windows Update: 🛑 STOPPED (Idle)"
			if hSCM, _, _ := procOpenSCManager.Call(0, 0, 1); hSCM != 0 {
				svcName, _ := syscall.UTF16PtrFromString("wuauserv")
				if hSvc, _, _ := procOpenService.Call(hSCM, uintptr(unsafe.Pointer(svcName)), 4); hSvc != 0 {
					var status SERVICE_STATUS
					if ret, _, _ := procQueryServiceStatus.Call(hSvc, uintptr(unsafe.Pointer(&status))); ret != 0 {
						if status.dwCurrentState == 4 {
							netUpdateText = "Windows Update: ⚠️ RUNNING (Downloading/Installing)"
						}
					}
					procCloseServiceHandle.Call(hSvc)
				}
				procCloseServiceHandle.Call(hSCM)
			}

			cmdNetstat := exec.Command("cmd", "/c", "netstat -ano | findstr ESTABLISHED")
			cmdNetstat.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			out, err := cmdNetstat.Output()
			if err == nil {
				pidCounts := make(map[string]int)
				for _, line := range strings.Split(string(out), "\n") {
					fields := strings.Fields(line)
					if len(fields) >= 5 {
						pidCounts[fields[len(fields)-1]]++
					}
				}
				type kv struct {
					Key   string
					Value int
				}
				var ss []kv
				for k, v := range pidCounts {
					ss = append(ss, kv{k, v})
				}
				sort.Slice(ss, func(i, j int) bool { return ss[i].Value > ss[j].Value })

				var topApps []string
				for i := 0; i < len(ss) && i < 3; i++ {
					pid, count := ss[i].Key, ss[i].Value
					appName := getProcessNameNative(pid)
					if appName != "Unknown" {
						topApps = append(topApps, fmt.Sprintf("• %s (%d connections)", appName, count))
					}
				}
				if len(topApps) > 0 {
					netTopAppsText = strings.Join(topApps, "\n")
				} else {
					netTopAppsText = "No active external connections found."
				}
			}
			time.Sleep(5 * time.Second)
		}
	}()
}

func getSystemVolumeNative() int {
	procCoInitialize.Call(0)
	defer procCoUninitialize.Call()

	var enumerator uintptr
	ret, _, _ := procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&CLSID_MMDeviceEnumerator)),
		0, 23,
		uintptr(unsafe.Pointer(&IID_IMMDeviceEnumerator)),
		uintptr(unsafe.Pointer(&enumerator)))
	if ret != 0 || enumerator == 0 {
		return -1
	}
	defer syscall.SyscallN((*[3]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(enumerator))))[2], enumerator)

	var device uintptr
	ret, _, _ = syscall.SyscallN((*[10]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(enumerator))))[4], enumerator, 0, 1, uintptr(unsafe.Pointer(&device)))
	if ret != 0 || device == 0 {
		return -1
	}
	defer syscall.SyscallN((*[3]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(device))))[2], device)

	var audioEndpointVolume uintptr
	ret, _, _ = syscall.SyscallN((*[10]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(device))))[3], device, uintptr(unsafe.Pointer(&IID_IAudioEndpointVolume)), 23, 0, uintptr(unsafe.Pointer(&audioEndpointVolume)))
	if ret != 0 || audioEndpointVolume == 0 {
		return -1
	}
	defer syscall.SyscallN((*[3]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(audioEndpointVolume))))[2], audioEndpointVolume)

	var vol float32
	syscall.SyscallN((*[15]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(audioEndpointVolume))))[9], audioEndpointVolume, uintptr(unsafe.Pointer(&vol)))

	return int(vol * 100.0)
}

func setSystemVolumeNatively(target int) {
	if target < 0 {
		target = 0
	}
	if target > 100 {
		target = 100
	}

	for i := 0; i < 50; i++ {
		curr := getSystemVolumeNative()
		if curr == -1 || curr == target {
			break
		}

		diff := curr - target
		if diff < 0 {
			diff = -diff
		}
		if diff <= 1 {
			break
		}

		vk := VK_VOLUME_UP
		if curr > target {
			vk = VK_VOLUME_DOWN
		}
		procKeybdEvent.Call(uintptr(vk), 0, 0, 0)
		procKeybdEvent.Call(uintptr(vk), 0, 2, 0)
		time.Sleep(5 * time.Millisecond)
	}
}

func updatePowerPlanStatus() {
	var schemeGuid *syscall.GUID
	ret, _, _ := procPowerGetActiveScheme.Call(0, uintptr(unsafe.Pointer(&schemeGuid)))
	if ret == 0 && schemeGuid != nil {
		sysPowerPlanGUID = fmt.Sprintf("%08x-%04x-%04x-%02x%02x-%02x%02x%02x%02x%02x%02x",
			schemeGuid.Data1, schemeGuid.Data2, schemeGuid.Data3,
			schemeGuid.Data4[0], schemeGuid.Data4[1], schemeGuid.Data4[2], schemeGuid.Data4[3],
			schemeGuid.Data4[4], schemeGuid.Data4[5], schemeGuid.Data4[6], schemeGuid.Data4[7])

		var bufferSize uint32
		procPowerReadFriendlyName.Call(0, uintptr(unsafe.Pointer(schemeGuid)), 0, 0, 0, uintptr(unsafe.Pointer(&bufferSize)))
		if bufferSize > 0 {
			buf := make([]uint16, bufferSize/2)
			if r, _, _ := procPowerReadFriendlyName.Call(0, uintptr(unsafe.Pointer(schemeGuid)), 0, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufferSize))); r == 0 {
				sysPowerPlan = syscall.UTF16ToString(buf)
			}
		}
		procLocalFree.Call(uintptr(unsafe.Pointer(schemeGuid)))
	}
}

func startToolsScanner() {
	go func() {
		for {
			if (isAsus && currentPage == 4) || (!isAsus && currentPage == 3) {
				if v := getSystemVolumeNative(); v != -1 && !isDraggingVolume {
					sysVolumeVal = v
				}

				if isLaptop && !isDraggingBrightness {
					cmdBr := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction SilentlyContinue).CurrentBrightness")
					cmdBr.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
					if outBr, err := cmdBr.Output(); err == nil {
						if v, err := strconv.Atoi(strings.TrimSpace(string(outBr))); err == nil {
							sysBrightnessVal = v
						}
					}
				}
				procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
			}

			if isLaptop {
				var status struct {
					ACLineStatus, BatteryFlag, BatteryLifePercent, Reserved1 byte
					BatteryLifeTime, BatteryFullLifeTime                     uint32
				}
				if ret, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status))); ret != 0 {
					batteryLevel = fmt.Sprintf("%.1f%%", float64(status.BatteryLifePercent))
					if status.ACLineStatus == 1 {
						sysBatteryPercent = fmt.Sprintf("%d%% (Plugged In)", status.BatteryLifePercent)
					} else {
						sysBatteryPercent = fmt.Sprintf("%d%%", status.BatteryLifePercent)
					}
				}
			} else {
				sysBatteryPercent = "AC Power (Desktop PC)"
				batteryLevel = "N/A"
				tick, _, _ := procGetTickCount64.Call()
				uptimeSec := tick / 1000
				hours := uptimeSec / 3600
				mins := (uptimeSec % 3600) / 60
				sysUptime = fmt.Sprintf("%dh %dm", hours, mins)
			}

			if atomic.LoadUint32(&loadingButtonID) == 0 {
				updatePowerPlanStatus()
			}

			if isAsus && currentPage == 2 {
				procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
			}

			time.Sleep(2 * time.Second)
		}
	}()
}
