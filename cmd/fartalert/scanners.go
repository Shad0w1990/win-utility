package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
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

	var status struct {
		ACLineStatus, BatteryFlag, BatteryLifePercent, Reserved1 byte
		BatteryLifeTime, BatteryFullLifeTime                     uint32
	}
	ret, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status)))
	hasBattery := (ret != 0 && status.BatteryFlag != 128)

	if strings.Contains(strings.ToUpper(sysModelName), "ASUS") || hasBattery {
		isAsusLaptop = true
	}

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

// ذخیره هوشمند سابقه سخت افزار تا یک ماه
func startTelemetryLogger() {
	appData := os.Getenv("LOCALAPPDATA")
	dir := filepath.Join(appData, "SysGuard")
	os.MkdirAll(dir, 0755)
	telemetryFile := filepath.Join(dir, "telemetry_1month.json")

	// لود کردن سابقه قبلی در صورت وجود
	data, err := os.ReadFile(telemetryFile)
	if err == nil {
		json.Unmarshal(data, &telemetryHistory)
	}

	// پاک کردن دیتای قدیمی‌تر از 30 روز
	thirtyDaysAgo := time.Now().Unix() - (30 * 24 * 60 * 60)
	var filtered []TelemetryData
	for _, t := range telemetryHistory {
		if t.Timestamp > thirtyDaysAgo {
			filtered = append(filtered, t)
		}
	}
	telemetryHistory = filtered

	// ضبط سابقه هر 10 دقیقه
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
	startTelemetryLogger() // استارت سیستم مانیتورینگ ماهانه

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
		for {
			var state XINPUT_STATE
			ret, _, _ := procXInputGetState.Call(0, uintptr(unsafe.Pointer(&state)))

			if ret == 0 {
				if sysGamepadName == "Searching..." {
					query := `(Get-PnpDevice -PresentOnly -ErrorAction SilentlyContinue | Where-Object { $_.FriendlyName -match 'Xbox|Controller|Gamepad|DualSense|DualShock' -and $_.Class -ne 'AudioEndpoint' -and $_.Class -ne 'Audio' } | Select-Object -First 1).FriendlyName`
					name := getWmiInfo(query)
					if name != "" {
						sysGamepadName = name
					} else {
						sysGamepadName = "Standard XInput Controller"
					}
				}
			} else {
				sysGamepadName = "Searching..."
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

func fetchLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Offline"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "Offline"
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

				cmdTasklist := exec.Command("cmd", "/c", "tasklist /FO CSV /NH")
				cmdTasklist.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				taskOut, _ := cmdTasklist.Output()

				var topApps []string
				for i := 0; i < len(ss) && i < 3; i++ {
					pid, count, appName := ss[i].Key, ss[i].Value, "Unknown"
					for _, tLine := range strings.Split(string(taskOut), "\n") {
						if strings.Contains(tLine, "\""+pid+"\"") {
							if parts := strings.Split(tLine, "\",\""); len(parts) > 0 {
								appName = strings.Trim(parts[0], "\"")
								break
							}
						}
					}
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

func startToolsScanner() {
	go func() {
		for {
			var status struct {
				ACLineStatus, BatteryFlag, BatteryLifePercent, Reserved1 byte
				BatteryLifeTime, BatteryFullLifeTime                     uint32
			}

			if ret, _, _ := procGetSystemPowerStatus.Call(uintptr(unsafe.Pointer(&status))); ret != 0 {
				if status.BatteryFlag != 128 && status.BatteryLifePercent <= 100 {
					batteryLevel = fmt.Sprintf("%.1f%%", float64(status.BatteryLifePercent))
					sysBatteryPercent = fmt.Sprintf("%d%%", status.BatteryLifePercent)

					if status.ACLineStatus == 1 {
						sysBatteryPercent += " (Plugged In)"
					}
				} else {
					sysBatteryPercent = "Desktop PC (No Battery)"
					batteryLevel = "N/A"
				}
			}

			cmdPower := exec.Command("cmd", "/c", "powercfg /getactivescheme")
			cmdPower.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			if out, err := cmdPower.Output(); err == nil {
				outStr := string(out)
				if strings.Contains(outStr, "(") && strings.Contains(outStr, ")") {
					start := strings.Index(outStr, "(") + 1
					end := strings.Index(outStr, ")")
					if start < end {
						sysPowerPlan = outStr[start:end]
					}
				}
			}

			if isAsusLaptop {
				cmdBright := exec.Command("powershell", "-NoProfile", "-Command", "(Get-CimInstance -Namespace root/WMI -ClassName WmiMonitorBrightness -ErrorAction SilentlyContinue).CurrentBrightness")
				cmdBright.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				if out, err := cmdBright.Output(); err == nil {
					res := strings.TrimSpace(string(out))
					if res != "" {
						sysBrightness = res + "%"
					} else {
						sysBrightness = "External / NA"
					}
				}
			} else {
				sysBrightness = "PC / External Monitor"
			}

			sysVolume = "System Audio Active"

			if (isAsusLaptop && currentPage == 4) || (!isAsusLaptop && currentPage == 3) {
				procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
			}
			if isAsusLaptop && currentPage == 2 {
				procInvalidateRect.Call(uintptr(hwndMain), 0, 1)
			}

			time.Sleep(8 * time.Second)
		}
	}()
}
