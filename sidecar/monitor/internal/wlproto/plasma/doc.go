//go:build linux

// Package plasma contains Go bindings for KDE's org_kde_plasma_window_management
// protocol, used as a KDE-native fallback to detect the active window on
// KWin/Plasma when the wlr protocol is unavailable.
//
// The *.xml.go file is generated from the vendored protocol XML with
// neurlang's wayland-scanner. To regenerate after updating the XML, run
// `go generate ./...` from this directory and re-add the `//go:build linux`
// tag to the generated file. types.go supplies the core-type aliases and
// SafeCast wrapper the generated code expects.
package plasma

//go:generate go run github.com/neurlang/wayland/cmd/wayland-scanner -i plasma-window-management.xml
//go:generate gofmt -w plasma-window-management.xml.go
