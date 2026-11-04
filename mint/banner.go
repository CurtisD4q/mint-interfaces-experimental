package mint

import (
	"strings"
	"unicode/utf8"
)

// Banner draws text as a big title: a 3×5 pixel font packed two pixel rows
// to a terminal row with the half-block characters ▀ ▄ █, so each letter is
// three columns wide and the whole banner is BannerHeight rows tall.
//
//	█▄█ ▀█▀ █▀▄ ▀█▀
//	█▀█  █  █ █  █
//	▀ ▀ ▀▀▀ ▀ ▀  ▀
//
// The font has A–Z (lower case draws as upper case), 0–9, space and the
// marks . , - _ : ! ? ' / +. Any other character draws as ?. There is one
// blank column between letters, so a banner of n characters is 4n-1 cells
// wide (see BannerWidth). Every line is that wide, padded with spaces.
//
// Banner only builds the lines. Set App.BannerText to show one at the top of
// the home list, or draw the lines yourself. (Screen.Banner is something
// else: the amber notice line under a screen's title.)
func Banner(text string) []string {
	rs := []rune(text)
	if len(rs) == 0 {
		return nil
	}
	w := 4*len(rs) - 1

	// Five rows of pixels, plus a blank sixth so they pair up into three rows.
	var px [2 * BannerHeight][]bool
	for y := range px {
		px[y] = make([]bool, w)
	}
	for i, r := range rs {
		for y, row := range strings.Split(bannerGlyph(r), "|") {
			for x, c := range row {
				if c == 'X' {
					px[y][4*i+x] = true
				}
			}
		}
	}

	lines := make([]string, BannerHeight)
	for l := range lines {
		var b strings.Builder
		for x := 0; x < w; x++ {
			top, bottom := px[2*l][x], px[2*l+1][x]
			switch {
			case top && bottom:
				b.WriteString("█")
			case top:
				b.WriteString("▀")
			case bottom:
				b.WriteString("▄")
			default:
				b.WriteByte(' ')
			}
		}
		lines[l] = b.String()
	}
	return lines
}

// BannerHeight is the number of terminal rows a Banner takes.
const BannerHeight = 3

// BannerWidth is the number of terminal columns Banner(text) takes: 4 for
// each character, less the gap after the last one. It is 0 for empty text.
func BannerWidth(text string) int {
	n := utf8.RuneCountInString(text)
	if n == 0 {
		return 0
	}
	return 4*n - 1
}

// bannerStyle colours a banner like a screen title.
func bannerStyle() string { return C.Accent + C.Bold }

// bannerFits reports whether the home screen has room for a banner: it must
// be allowed (UTF-8 terminal), fit across the frame, and still leave room for
// two home items below it.
func bannerFits(h, w int, text string, noteRows int, asciiOnly bool) bool {
	if text == "" || asciiOnly {
		return false
	}
	if BannerWidth(text) > w-6 {
		return false
	}
	listTop := 4 + BannerHeight + 1 + noteRows
	contentBottom := h - 2
	return contentBottom-listTop+1 >= 2*homeBlock
}

// homeHeaderRows lays out the rows between the title rule (row 3) and the
// home list. It returns the banner lines to draw from row 4 (nil when the
// banner is off or does not fit), the row where the note starts, and the row
// where the list starts. With no banner the list starts at row 4 plus the
// note's rows, as it always has.
func homeHeaderRows(h, w int, banner string, noteRows int, asciiOnly bool) (lines []string, noteTop, contentTop int) {
	top := 4
	if bannerFits(h, w, banner, noteRows, asciiOnly) {
		lines = Banner(banner)
		top += len(lines) + 1 // the banner and one blank row under it
	}
	return lines, top, top + noteRows
}

// bannerGlyphs is the font: five rows of three pixels, X lit, rows joined by |.
var bannerGlyphs = map[rune]string{
	'A': ".X.|X.X|XXX|X.X|X.X", 'B': "XX.|X.X|XX.|X.X|XX.", 'C': ".XX|X..|X..|X..|.XX",
	'D': "XX.|X.X|X.X|X.X|XX.", 'E': "XXX|X..|XX.|X..|XXX", 'F': "XXX|X..|XX.|X..|X..",
	'G': ".XX|X..|X.X|X.X|.XX", 'H': "X.X|X.X|XXX|X.X|X.X", 'I': "XXX|.X.|.X.|.X.|XXX",
	'J': "..X|..X|..X|X.X|.X.", 'K': "X.X|X.X|XX.|X.X|X.X", 'L': "X..|X..|X..|X..|XXX",
	'M': "X.X|XXX|XXX|X.X|X.X", 'N': "XX.|X.X|X.X|X.X|X.X", 'O': ".X.|X.X|X.X|X.X|.X.",
	'P': "XX.|X.X|XX.|X..|X..", 'Q': ".X.|X.X|X.X|XXX|.XX", 'R': "XX.|X.X|XX.|X.X|X.X",
	'S': ".XX|X..|.X.|..X|XX.", 'T': "XXX|.X.|.X.|.X.|.X.", 'U': "X.X|X.X|X.X|X.X|XXX",
	'V': "X.X|X.X|X.X|X.X|.X.", 'W': "X.X|X.X|XXX|XXX|X.X", 'X': "X.X|X.X|.X.|X.X|X.X",
	'Y': "X.X|X.X|.X.|.X.|.X.", 'Z': "XXX|..X|.X.|X..|XXX",

	'0': "XXX|X.X|X.X|X.X|XXX", '1': ".X.|XX.|.X.|.X.|XXX", '2': "XX.|..X|.X.|X..|XXX",
	'3': "XX.|..X|.X.|..X|XX.", '4': "X.X|X.X|XXX|..X|..X", '5': "XXX|X..|XX.|..X|XX.",
	'6': ".XX|X..|XXX|X.X|XXX", '7': "XXX|..X|.X.|.X.|.X.", '8': "XXX|X.X|XXX|X.X|XXX",
	'9': "XXX|X.X|XXX|..X|XX.",

	' ': "...|...|...|...|...",
	'.': "...|...|...|...|.X.", ',': "...|...|...|.X.|X..",
	'-': "...|...|XXX|...|...", '_': "...|...|...|...|XXX",
	':': "...|.X.|...|.X.|...", '!': ".X.|.X.|.X.|...|.X.",
	'?': "XX.|..X|.X.|...|.X.", '\'': ".X.|.X.|...|...|...",
	'/': "..X|..X|.X.|X..|X..", '+': "...|.X.|XXX|.X.|...",
}

// bannerGlyph returns r's glyph; lower case is upper-cased, and anything the
// font lacks draws as ?.
func bannerGlyph(r rune) string {
	if r >= 'a' && r <= 'z' {
		r -= 'a' - 'A'
	}
	if g, ok := bannerGlyphs[r]; ok {
		return g
	}
	return bannerGlyphs['?']
}
