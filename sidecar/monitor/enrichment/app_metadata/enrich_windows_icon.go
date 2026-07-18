//go:build windows

package appmetadata

import (
	"image"
	"image/png"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Win32 bindings needed to turn an executable's icon into a PNG. Kept local to
// this file since x/sys/windows does not wrap ExtractIconEx / GDI DIB reads.
var (
	modShell32 = windows.NewLazySystemDLL("shell32.dll")
	modUser32  = windows.NewLazySystemDLL("user32.dll")
	modGDI32   = windows.NewLazySystemDLL("gdi32.dll")

	procExtractIconExW = modShell32.NewProc("ExtractIconExW")
	procGetIconInfo    = modUser32.NewProc("GetIconInfo")
	procDestroyIcon    = modUser32.NewProc("DestroyIcon")
	procGetDC          = modUser32.NewProc("GetDC")
	procReleaseDC      = modUser32.NewProc("ReleaseDC")
	procGetObjectW     = modGDI32.NewProc("GetObjectW")
	procGetDIBits      = modGDI32.NewProc("GetDIBits")
	procDeleteObject   = modGDI32.NewProc("DeleteObject")
)

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  windows.Handle
	hbmColor windows.Handle
}

type bitmap struct {
	bmType       int32
	bmWidth      int32
	bmHeight     int32
	bmWidthBytes int32
	bmPlanes     uint16
	bmBitsPixel  uint16
	bmBits       uintptr
}

type bitmapInfoHeader struct {
	biSize          uint32
	biWidth         int32
	biHeight        int32
	biPlanes        uint16
	biBitCount      uint16
	biCompression   uint32
	biSizeImage     uint32
	biXPelsPerMeter int32
	biYPelsPerMeter int32
	biClrUsed       uint32
	biClrImportant  uint32
}

// extractWindowsIcon pulls the first large icon out of an executable and writes
// it to the icon cache as a PNG. Best-effort: any failure returns "" so
// enrichment continues without an icon.
func extractWindowsIcon(exe string, identity string) string {
	target, err := iconCachePath(identity, "png")
	if err != nil {
		return ""
	}
	if _, err := os.Stat(target); err == nil {
		return target // already cached
	}

	img, ok := iconImageFromExe(exe)
	if !ok {
		return ""
	}

	out, err := os.Create(target)
	if err != nil {
		return ""
	}
	defer out.Close()
	if err := png.Encode(out, img); err != nil {
		os.Remove(target)
		return ""
	}
	return target
}

func iconImageFromExe(exe string) (image.Image, bool) {
	path, err := windows.UTF16PtrFromString(exe)
	if err != nil {
		return nil, false
	}

	var hIcon windows.Handle
	// ExtractIconExW(path, index=0, largeIcons*, smallIcons*, count=1) -> extracted count.
	ret, _, _ := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(path)),
		0,
		uintptr(unsafe.Pointer(&hIcon)),
		0,
		1,
	)
	if ret == 0 || hIcon == 0 {
		return nil, false
	}
	defer procDestroyIcon.Call(uintptr(hIcon))

	var info iconInfo
	if r, _, _ := procGetIconInfo.Call(uintptr(hIcon), uintptr(unsafe.Pointer(&info))); r == 0 {
		return nil, false
	}
	if info.hbmColor != 0 {
		defer procDeleteObject.Call(uintptr(info.hbmColor))
	}
	if info.hbmMask != 0 {
		defer procDeleteObject.Call(uintptr(info.hbmMask))
	}
	if info.hbmColor == 0 {
		return nil, false
	}

	var bmp bitmap
	if r, _, _ := procGetObjectW.Call(
		uintptr(info.hbmColor),
		unsafe.Sizeof(bmp),
		uintptr(unsafe.Pointer(&bmp)),
	); r == 0 {
		return nil, false
	}
	width, height := int(bmp.bmWidth), int(bmp.bmHeight)
	if width <= 0 || height <= 0 || width > 1024 || height > 1024 {
		return nil, false
	}

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, false
	}
	defer procReleaseDC.Call(0, hdc)

	header := bitmapInfoHeader{
		biSize:        uint32(unsafe.Sizeof(bitmapInfoHeader{})),
		biWidth:       int32(width),
		biHeight:      -int32(height), // negative => top-down rows
		biPlanes:      1,
		biBitCount:    32,
		biCompression: 0, // BI_RGB
	}
	pixels := make([]byte, width*height*4)
	// GetDIBits(hdc, hbm, startScan=0, cLines=height, bits, bmi, usage=DIB_RGB_COLORS).
	if r, _, _ := procGetDIBits.Call(
		hdc,
		uintptr(info.hbmColor),
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&pixels[0])),
		uintptr(unsafe.Pointer(&header)),
		0,
	); r == 0 {
		return nil, false
	}

	return bgraToImage(pixels, width, height), true
}

// bgraToImage converts a top-down BGRA pixel buffer into an NRGBA image. Icons
// whose color bitmap carries no alpha (all zero) are treated as fully opaque.
func bgraToImage(pixels []byte, width int, height int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	anyAlpha := false
	for i := 3; i < len(pixels); i += 4 {
		if pixels[i] != 0 {
			anyAlpha = true
			break
		}
	}
	for i := 0; i+3 < len(pixels); i += 4 {
		b, g, r, a := pixels[i], pixels[i+1], pixels[i+2], pixels[i+3]
		if !anyAlpha {
			a = 0xff
		}
		img.Pix[i] = r
		img.Pix[i+1] = g
		img.Pix[i+2] = b
		img.Pix[i+3] = a
	}
	return img
}
