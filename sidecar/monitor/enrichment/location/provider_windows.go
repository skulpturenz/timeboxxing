//go:build windows

package location

import (
	"context"
	"log/slog"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/skulpturenz/timeboxxing/sidecar/monitor/permission"
)

// Windows location capture via the WinRT Windows.Devices.Geolocation.Geolocator.
//
// Driven through raw COM vtable calls over golang.org/x/sys/windows syscalls
// (cgo-free). A dedicated OS-thread goroutine initializes the multithreaded
// apartment, activates a Geolocator, requests location access, and polls the
// position on a long interval, caching each fix under a lock. Location()/
// Permission() read the cache non-blockingly.
//
// NOTE: the IIDs and vtable indices below are canonical Windows SDK values but
// have not been runtime-verified on a Windows host — a wrong constant makes a
// COM call fail and degrades the provider to "no fix" (never a crash).

var (
	combase                    = windows.NewLazySystemDLL("combase.dll")
	procRoInitialize           = combase.NewProc("RoInitialize")
	procRoActivateInstance     = combase.NewProc("RoActivateInstance")
	procRoGetActivationFactory = combase.NewProc("RoGetActivationFactory")
	procWindowsCreateString    = combase.NewProc("WindowsCreateString")
	procWindowsDeleteString    = combase.NewProc("WindowsDeleteString")
)

const (
	roInitMultithreaded = 1

	asyncStatusStarted   = 0
	asyncStatusCompleted = 1

	geolocationAccessAllowed = 1

	geolocatorClassName = "Windows.Devices.Geolocation.Geolocator"
)

// IUnknown / IInspectable and interface-specific vtable slot indices.
const (
	vtQueryInterface = 0
	vtRelease        = 2

	vtGeolocatorGetGeopositionAsync = 13 // IGeolocator
	vtStaticsRequestAccessAsync     = 6  // IGeolocatorStatics
	vtAsyncInfoGetStatus            = 7  // IAsyncInfo
	vtAsyncOperationGetResults      = 8  // IAsyncOperation<T>
	vtGeopositionGetCoordinate      = 6  // IGeoposition
	vtGeocoordinateGetLatitude      = 6  // IGeocoordinate (deprecated: doubles by out-param)
	vtGeocoordinateGetLongitude     = 7  // IGeocoordinate
)

var (
	iidIGeolocator        = windows.GUID{Data1: 0xA9C3BF62, Data2: 0x4524, Data3: 0x4989, Data4: [8]byte{0x8A, 0xA9, 0xDE, 0x01, 0x9D, 0x2E, 0x55, 0x1F}}
	iidIGeolocatorStatics = windows.GUID{Data1: 0x5DA253EE, Data2: 0x6B79, Data3: 0x45E1, Data4: [8]byte{0x8E, 0x0B, 0xDC, 0x01, 0x84, 0xF9, 0xF2, 0xD9}}
	iidIAsyncInfo         = windows.GUID{Data1: 0x00000036, Data2: 0x0000, Data3: 0x0000, Data4: [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
	iidIGeocoordinate     = windows.GUID{Data1: 0x642E4B6B, Data2: 0xCFA9, Data3: 0x4EAD, Data4: [8]byte{0xBB, 0x98, 0xEF, 0x7F, 0x71, 0xAB, 0xE4, 0x02}}
)

var ptrSize = int(unsafe.Sizeof(uintptr(0)))

// windowsLocationProvider caches the latest WinRT Geolocator fix and access
// grant. Satisfies LocationProvider.
type windowsLocationProvider struct {
	mu       sync.RWMutex
	lat, lon float64
	hasFix   bool
	granted  bool
}

// DefaultLocationProvider starts the background WinRT location provider and
// returns a provider that reads its cache non-blockingly.
func DefaultLocationProvider() LocationProvider {
	p := &windowsLocationProvider{}
	go p.run()
	return p
}

func (p *windowsLocationProvider) Location() (float64, float64, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if !p.hasFix {
		return 0, 0, false
	}
	return p.lat, p.lon, true
}

func (p *windowsLocationProvider) Permission() (permission.Status, bool) {
	p.mu.RLock()
	granted := p.granted
	p.mu.RUnlock()
	return permission.Status{
		Name:       "Location Services",
		Granted:    granted,
		HowToGrant: "Settings → Privacy & security → Location → enable location access for this app",
	}, true
}

// RequestPermission raises the Windows location-access prompt via the Geolocator
// static factory's RequestAccessAsync, on its own COM-initialized thread. Deferred
// to this call (rather than the run() startup) so the caller controls when it fires.
func (p *windowsLocationProvider) RequestPermission(context.Context) {
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()

		if r, _, _ := procRoInitialize.Call(uintptr(roInitMultithreaded)); r != 0 {
			slog.Debug("RoInitialize returned non-zero", "hr", r)
		}
		className, err := createHString(geolocatorClassName)
		if err != nil {
			slog.Debug("WindowsCreateString failed", "error", err)
			return
		}
		defer deleteHString(className)
		p.requestAccess(className)
	}()
}

// comCall invokes vtable slot `index` on a COM interface pointer `this`.
// COM interface pointers are carried as unsafe.Pointer (they reference native,
// non-Go-managed memory) so the vtable dereference stays vet-clean.
func comCall(this unsafe.Pointer, index int, args ...uintptr) uintptr {
	if this == nil {
		return 0x80004003 // E_POINTER
	}
	vtbl := *(*unsafe.Pointer)(this)
	fn := *(*uintptr)(unsafe.Add(vtbl, index*ptrSize))
	all := make([]uintptr, 0, len(args)+1)
	all = append(all, uintptr(this))
	all = append(all, args...)
	ret, _, _ := syscall.SyscallN(fn, all...)
	return ret
}

func comRelease(this unsafe.Pointer) {
	if this != nil {
		comCall(this, vtRelease)
	}
}

func createHString(s string) (uintptr, error) {
	u16, err := windows.UTF16FromString(s)
	if err != nil {
		return 0, err
	}
	var h uintptr
	// length excludes the terminating NUL.
	r, _, _ := procWindowsCreateString.Call(
		uintptr(unsafe.Pointer(&u16[0])),
		uintptr(len(u16)-1),
		uintptr(unsafe.Pointer(&h)),
	)
	if r != 0 {
		return 0, syscall.Errno(r)
	}
	return h, nil
}

func deleteHString(h uintptr) {
	if h != 0 {
		procWindowsDeleteString.Call(h)
	}
}

// awaitAsync polls IAsyncInfo::Status until the operation completes or the
// deadline passes, returning whether it completed successfully.
func awaitAsync(asyncOp unsafe.Pointer, timeout time.Duration) bool {
	var asyncInfo unsafe.Pointer
	if comCall(asyncOp, vtQueryInterface, uintptr(unsafe.Pointer(&iidIAsyncInfo)), uintptr(unsafe.Pointer(&asyncInfo))) != 0 || asyncInfo == nil {
		return false
	}
	defer comRelease(asyncInfo)

	deadline := time.Now().Add(timeout)
	for {
		var status int32
		if comCall(asyncInfo, vtAsyncInfoGetStatus, uintptr(unsafe.Pointer(&status))) != 0 {
			return false
		}
		switch status {
		case asyncStatusCompleted:
			return true
		case asyncStatusStarted:
			// still running
		default:
			return false // canceled or error
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (p *windowsLocationProvider) run() {
	// COM apartment state is per-thread, so pin this goroutine.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if r, _, _ := procRoInitialize.Call(uintptr(roInitMultithreaded)); r != 0 {
		// S_FALSE / already-initialized are non-fatal; a hard failure will
		// surface as a failed activation below.
		slog.Debug("RoInitialize returned non-zero", "hr", r)
	}

	className, err := createHString(geolocatorClassName)
	if err != nil {
		slog.Debug("WindowsCreateString failed", "error", err)
		return
	}
	defer deleteHString(className)

	var inspectable unsafe.Pointer
	if r, _, _ := procRoActivateInstance.Call(className, uintptr(unsafe.Pointer(&inspectable))); r != 0 || inspectable == nil {
		slog.Debug("RoActivateInstance(Geolocator) failed", "hr", r)
		return
	}
	var geolocator unsafe.Pointer
	if comCall(inspectable, vtQueryInterface, uintptr(unsafe.Pointer(&iidIGeolocator)), uintptr(unsafe.Pointer(&geolocator))) != 0 || geolocator == nil {
		comRelease(inspectable)
		return
	}
	comRelease(inspectable)
	defer comRelease(geolocator)

	for {
		p.fetchOnce(geolocator)
		time.Sleep(5 * time.Minute)
	}
}

func (p *windowsLocationProvider) requestAccess(className uintptr) {
	var factory unsafe.Pointer
	if r, _, _ := procRoGetActivationFactory.Call(className, uintptr(unsafe.Pointer(&iidIGeolocatorStatics)), uintptr(unsafe.Pointer(&factory))); r != 0 || factory == nil {
		return
	}
	defer comRelease(factory)

	var asyncOp unsafe.Pointer
	if comCall(factory, vtStaticsRequestAccessAsync, uintptr(unsafe.Pointer(&asyncOp))) != 0 || asyncOp == nil {
		return
	}
	defer comRelease(asyncOp)

	if !awaitAsync(asyncOp, 30*time.Second) {
		return
	}
	var status int32
	if comCall(asyncOp, vtAsyncOperationGetResults, uintptr(unsafe.Pointer(&status))) != 0 {
		return
	}
	p.mu.Lock()
	p.granted = status == geolocationAccessAllowed
	p.mu.Unlock()
}

func (p *windowsLocationProvider) fetchOnce(geolocator unsafe.Pointer) {
	var asyncOp unsafe.Pointer
	if comCall(geolocator, vtGeolocatorGetGeopositionAsync, uintptr(unsafe.Pointer(&asyncOp))) != 0 || asyncOp == nil {
		return
	}
	defer comRelease(asyncOp)

	if !awaitAsync(asyncOp, 30*time.Second) {
		return
	}

	var geoposition unsafe.Pointer
	if comCall(asyncOp, vtAsyncOperationGetResults, uintptr(unsafe.Pointer(&geoposition))) != 0 || geoposition == nil {
		return
	}
	defer comRelease(geoposition)

	var coordinate unsafe.Pointer
	if comCall(geoposition, vtGeopositionGetCoordinate, uintptr(unsafe.Pointer(&coordinate))) != 0 || coordinate == nil {
		return
	}
	defer comRelease(coordinate)

	var lat, lon float64
	if comCall(coordinate, vtGeocoordinateGetLatitude, uintptr(unsafe.Pointer(&lat))) != 0 {
		return
	}
	if comCall(coordinate, vtGeocoordinateGetLongitude, uintptr(unsafe.Pointer(&lon))) != 0 {
		return
	}

	p.mu.Lock()
	p.lat, p.lon, p.hasFix = lat, lon, true
	p.mu.Unlock()
}
