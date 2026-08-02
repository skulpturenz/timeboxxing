//go:build ruleguard

package gorules

import "github.com/quasilyte/go-ruleguard/dsl"

func noChainedLiteralCall(m dsl.Matcher) {
	m.Match(`$t{$*_}.$method($*_)`).
		Report(`do not chain a call onto a struct literal: bind it to a variable, then call $method on the next line`)
}
