package enumsmaskingcategory

import (
	"fmt"
	"strings"
)

// MaskingCategory scopes a masking table lookup, so the same text as an app name and as a domain
// masks to two unrelated tokens. Despite the name, unrelated to the application category taxonomy
// and to the masked_categories table that permutes it.
type MaskingCategory int

const (
	Unknown MaskingCategory = iota
	AppName
	AppIdentifier
	AppFriendlyName
	BrowserVendor
	BrowserAppIdentifier
	BrowserDomain
)

func (maskingCategory MaskingCategory) String() string {
	return []string{
		"unknown",
		"app_name",
		"app_identifier",
		"app_friendly_name",
		"browser_vendor",
		"browser_app_identifier",
		"browser_domain",
	}[maskingCategory]
}

func Parse(code string) (MaskingCategory, error) {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "app_name":
		return AppName, nil
	case "app_identifier":
		return AppIdentifier, nil
	case "app_friendly_name":
		return AppFriendlyName, nil
	case "browser_vendor":
		return BrowserVendor, nil
	case "browser_app_identifier":
		return BrowserAppIdentifier, nil
	case "browser_domain":
		return BrowserDomain, nil
	}

	return Unknown, fmt.Errorf("unrecognized masking category: %s", code)
}
