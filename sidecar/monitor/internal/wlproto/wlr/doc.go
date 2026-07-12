//go:build linux

// Package wlr contains Go bindings for the wlr-foreign-toplevel-management
// unstable protocol, used to detect the active (activated) toplevel on
// wlroots-based compositors (Sway, Hyprland, river, Wayfire, niri, COSMIC,
// labwc/phoc/cage) and KWin.
//
// The *.xml.go file is generated from the vendored protocol XML with
// neurlang's wayland-scanner. To regenerate after updating the XML, run
// `go generate ./...` from this directory and re-add the `//go:build linux`
// tag to the generated file (the scanner does not emit it). types.go supplies
// the core-type aliases and SafeCast wrapper the generated code expects.
package wlr

//go:generate go run github.com/neurlang/wayland/cmd/wayland-scanner -i wlr-foreign-toplevel-management-unstable-v1.xml
//go:generate gofmt -w wlr-foreign-toplevel-management-unstable-v1.xml.go
