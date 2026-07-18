//go:build windows

package appmetadata

import (
	"context"
	"strings"
	"unsafe"

	sessionnew "github.com/skulpturenz/timeboxxing/sidecar/monitor/session_new"
	"golang.org/x/sys/windows"
)

func LocalMetadataEnricher(ctx context.Context, fp sessionnew.ForegroundProcess) (sessionnew.ForegroundProcess, bool) {
	if fp.AppPath == nil {
		return fp, false
	}
	exe := strings.TrimSpace(*fp.AppPath)
	if exe == "" {
		return fp, false
	}

	info, err := readVersionInfo(exe)
	metadata := Metadata{Source: SourcePE}
	if err == nil {
		metadata.FriendlyName = firstNonBlank(info.FileDescription, info.ProductName)
		metadata.Description = strings.TrimSpace(info.Comments)
	}
	if iconPath := extractWindowsIcon(exe, identityKey(fp)); iconPath != "" {
		metadata.IconPath = iconPath
	}

	if metadata.empty() {
		return fp, false
	}
	return setMetadata(fp, metadata)
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// versionInfo holds the PE version-resource string fields we consume.
type versionInfo struct {
	FileDescription string
	ProductName     string
	CompanyName     string
	Comments        string
}

// readVersionInfo loads the executable's version resource and reads the string
// table for the file's primary translation.
func readVersionInfo(path string) (versionInfo, error) {
	var zero windows.Handle
	size, err := windows.GetFileVersionInfoSize(path, &zero)
	if err != nil {
		return versionInfo{}, err
	}
	buffer := make([]byte, size)
	if err := windows.GetFileVersionInfo(path, 0, size, unsafe.Pointer(&buffer[0])); err != nil {
		return versionInfo{}, err
	}

	langCodepage := translationCode(buffer)

	read := func(field string) string {
		return verQueryString(buffer, langCodepage, field)
	}
	return versionInfo{
		FileDescription: read("FileDescription"),
		ProductName:     read("ProductName"),
		CompanyName:     read("CompanyName"),
		Comments:        read("Comments"),
	}, nil
}

// translationCode returns the "langID-codepage" hex string (e.g. "040904b0")
// identifying which StringFileInfo block to read. Falls back to US-English/
// Unicode when the translation table is missing.
func translationCode(block []byte) string {
	var ptr unsafe.Pointer
	var length uint32
	err := windows.VerQueryValue(
		unsafe.Pointer(&block[0]),
		`\VarFileInfo\Translation`,
		unsafe.Pointer(&ptr),
		&length,
	)
	if err != nil || length < 4 || ptr == nil {
		return "040904b0"
	}
	// The value is an array of {WORD language; WORD codePage}. Use the first.
	pair := (*[2]uint16)(ptr)
	return toHex16(pair[0]) + toHex16(pair[1])
}

func toHex16(v uint16) string {
	const digits = "0123456789abcdef"
	return string([]byte{
		digits[(v>>12)&0xf],
		digits[(v>>8)&0xf],
		digits[(v>>4)&0xf],
		digits[v&0xf],
	})
}

// verQueryString reads one string value out of the version resource's string
// table for the given translation.
func verQueryString(block []byte, langCodepage string, field string) string {
	subBlock := `\StringFileInfo\` + langCodepage + `\` + field
	var ptr unsafe.Pointer
	var length uint32
	err := windows.VerQueryValue(
		unsafe.Pointer(&block[0]),
		subBlock,
		unsafe.Pointer(&ptr),
		&length,
	)
	if err != nil || length == 0 || ptr == nil {
		return ""
	}
	// length is in UTF-16 code units, including the terminating NUL.
	utf16 := unsafe.Slice((*uint16)(ptr), length)
	for len(utf16) > 0 && utf16[len(utf16)-1] == 0 {
		utf16 = utf16[:len(utf16)-1]
	}
	return strings.TrimSpace(windows.UTF16ToString(utf16))
}
