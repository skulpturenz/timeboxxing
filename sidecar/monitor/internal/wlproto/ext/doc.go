//go:build linux

// Package ext contains Go bindings for the ext-idle-notify-v1 protocol, used
// to detect user idle/resume on Wayland compositors that implement it.
//
// The *.xml.go file is generated from the vendored protocol XML with
// neurlang's wayland-scanner. To regenerate after updating the XML, run
// `go generate ./...` from this directory and re-add the `//go:build linux`
// tag to the generated file. types.go supplies the core-type aliases and
// SafeCast wrapper the generated code expects.
package ext

//go:generate go run github.com/neurlang/wayland/cmd/wayland-scanner -i ext-idle-notify-v1.xml
//go:generate gofmt -w ext-idle-notify-v1.xml.go
