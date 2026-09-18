# 🌟 SysGuard Ultimate - Modular Edition

SysGuard Ultimate is a lightweight, blazing-fast Windows utility and hardware monitoring dashboard built entirely in **Go** using native **Win32 APIs**. It completely bypasses heavy web frameworks like Electron, offering real-time telemetry, advanced system management, and Zero-API AI diagnostics with a minimal system footprint.

## 🚀 Core Features (English)

*   📊 **Real-time Hardware Telemetry:** Native GDI-rendered graphs for CPU, RAM, GPU, and Disk usage. Logs historic data up to 7 days for deep bottleneck analysis.
*   🌐 **Advanced Network Analyzer:** Smart local IP detection (bypassing VMs/APIPA), live bandwidth graphs (DL/UL), and active connection monitoring (ESTABLISHED states).
*   ⚡ **System & Power Management:** Direct, real-time control over Windows Power Plans (Power Saver, Balanced, High Performance), native display brightness sliders (WMI), and volume controls (IAudioEndpointVolume).
*   🎮 **Pro Gamepad Tester:** Native XInput monitoring with real-time feedback for analog sticks, pressure triggers (LT/RT), and adaptive layout recognition (Xbox/PlayStation/Nintendo).
*   🧰 **Maintenance Suite:** Deep OS junk cleaner, DNS flush & Time sync, Audio driver wake/test, and comprehensive hardware status reports.
*   🧠 **Smart AI Diagnostics (Zero-API):**
    *   **Hardware Bottleneck Analyzer:** Evaluates 7-day historic telemetry to identify weak hardware components and suggest prioritized upgrades.
    *   **Smart Event Log Analyzer:** Natively extracts Level 1 & 2 (Critical/Error) Windows Event Logs. Generates structured prompts and seamlessly injects them into the clipboard for zero-cost, API-free Gemini AI diagnosis.
*   🎨 **Native & Smooth UI:** Fully responsive Win32 interface with dynamic localization (EN/FA), custom interactive "Info Cards," smooth dragging sliders, and pulse animations.

---

## 🚀 ویژگی‌های کلیدی (فارسی)

نرم‌افزار SysGuard Ultimate یک ابزار مانیتورینگ و بهینه‌سازی ویندوز است که کاملاً با زبان **Go** و توابع بومی **Win32 API** توسعه یافته است. این برنامه بدون نیاز به فریم‌ورک‌های سنگین مانند Electron کار می‌کند و با کمترین مصرف منابع، امکانات فوق‌پیشرفته‌ای از جمله خطایابی با هوش مصنوعی را ارائه می‌دهد.

*   📊 **داشبورد زنده سخت‌افزار:** مانیتورینگ دقیق و لحظه‌ای پردازنده، رم، گرافیک و هارد دیسک با گراف‌های اختصاصی GDI و ثبت دیتای ۷ روزه جهت تحلیل عملکرد سیستم.
*   🌐 **تحلیل‌گر پیشرفته شبکه:** فیلتر هوشمند IP واقعی سیستم (نادیده گرفتن ماشین‌های مجازی)، گراف زنده سرعت دانلود/آپلود و شناسایی نرم‌افزارهای پرمصرف اینترنت و وضعیت آپدیت ویندوز.
*   ⚡ **مدیریت انرژی و سیستم:** کنترل سریع پروفایل‌های مصرف انرژی ویندوز (Power Plans)، تنظیم مستقیم و روان روشنایی مانیتور و حجم صدای سیستم با استفاده از توابع بومی ویندوز.
*   🎮 **تستر حرفه‌ای گیم‌پد:** پشتیبانی از XInput برای مانیتورینگ زنده تریگرها، آنالوگ‌ها و دکمه‌ها با رابط کاربری تطبیق‌پذیر و فیدبک‌های بصری.
*   🧰 **ابزارهای نگهداری ویندوز:** پاکسازی عمیق فایل‌های موقت، همگام‌سازی اجباری ساعت و کش DNS، و تست موتور صوتی سیستم.
*   🧠 **عیب‌یابی هوشمند با هوش مصنوعی (Zero-API):**
    *   **آنالیز گلوگاه قطعات:** بررسی تلمتریِ گذشته سیستم برای پیدا کردن قطعه‌ای که باعث افت فریم یا کندی می‌شود.
    *   **تحلیل‌گر لاگ‌های پنهان:** استخراج لاگ‌های بحرانی (Critical/Error) از Event Viewer ویندوز و ساخت پرامپت‌های ساختاریافته. این پرامپت‌ها بدون نیاز به API Key مستقیماً در کلیپ‌بورد کپی شده و کاربر را برای دریافت راه‌حل دقیق، به هوش مصنوعی (Gemini) متصل می‌کنند.
*   🎨 **رابط کاربری بومی و روان:** طراحی ۱۰۰٪ بومی با انیمیشن‌های روان، قابلیت تغییر زبان زنده (فارسی/انگلیسی)، اسلایدرهای اختصاصی بدون لگ و کارت‌های اطلاعاتی تعاملی.

---

## ☕ Support the Developer / حمایت از توسعه‌دهنده

If you find this utility useful and want to support the project, you can buy me a coffee! ☕

اگر این ابزار برایتان مفید بوده و خواستید از توسعه پروژه حمایت کنید، می‌توانید از راه‌های زیر اقدام کنید:

*   **🇮🇷 شبکه شتاب (ایران):**
    *   شماره کارت: `6104-3377-6761-3068` (به نام احسان خرسند)
*   **🌐 Tether (USDT - TRC20 / Tron):**
    *   `TCzZtuWEwZfcWa3wKHGYjwHZrSPL6DW7C`
*   **💎 TON Network:**
    *   `UQB-5yLspFNXmvEXR4DP955To-D3hn2b0Rc3p7BNCqfzZAtF`

---

## 🛠️ Tech Stack & Architecture

- **Language:** Go (Golang)
- **UI Framework:** Pure Win32 API (No external dependencies, No CGO required for core UI)
- **Log Parsing:** Native PowerShell execution & WMI hooks
- **Architecture:** Concurrent goroutines for seamless background monitoring and fluid UI rendering.