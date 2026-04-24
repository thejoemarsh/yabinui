package tui

import "strings"

// Logo is the yabinui ASCII wordmark shown in the header.
var Logo = strings.Join([]string{
	"                __    _             _ ",
	"   __  ______ _/ /_  (_)___  __  __(_)",
	"  / / / / __ `/ __ \\/ / __ \\/ / / / / ",
	" / /_/ / /_/ / /_/ / / / / / /_/ / /  ",
	" \\__, /\\__,_/_.___/_/_/ /_/\\__,_/_/   ",
	"/____/                                ",
}, "\n")
