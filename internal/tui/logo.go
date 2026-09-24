package tui

import "strings"

var blocoLogoLines = []string{
	` _________  ____       _________  __________ _________`,
	`|     o   )/   /_____ /    O    \/   /_____//    O    \`,
	`|_____O___)\___\_____\\_________/\___\%%%%%'\_________/`,
	` 'BBBBBBB'  'BBBBBBBB' 'BBBBBBB'  'BBBBBBBB' 'BBBBBBB'`,
}

func renderBlocoLogo(pad string) string {
	var b strings.Builder
	for _, line := range blocoLogoLines {
		b.WriteString(pad)
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
