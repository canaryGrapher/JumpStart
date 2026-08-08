package analytics

import (
	"fmt"
	"syscall"
	"unsafe"
)

// readOSVersion returns the Windows version as "major.minor.build", e.g.
// "10.0.22631" (Windows 11 23H2).
//
// RtlGetNtVersionNumbers is used rather than GetVersionEx because the
// latter lies about the version unless the binary carries a compatibility
// manifest, which would make every Windows install look like 8.1.
func readOSVersion() string {
	defer func() {
		// Never let a version probe take the app down.
		_ = recover()
	}()

	proc := syscall.NewLazyDLL("ntdll.dll").NewProc("RtlGetNtVersionNumbers")
	if err := proc.Find(); err != nil {
		return "unknown"
	}

	var major, minor, build uint32
	proc.Call(
		uintptr(unsafe.Pointer(&major)),
		uintptr(unsafe.Pointer(&minor)),
		uintptr(unsafe.Pointer(&build)),
	)
	// The high bits are flags, not part of the build number.
	build &= 0xFFFF
	return fmt.Sprintf("%d.%d.%d", major, minor, build)
}
