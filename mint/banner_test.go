package mint

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestBannerGolden(t *testing.T) {
	want := []string{
		"█▄█ ▀█▀ █▀▄ ▀█▀",
		"█▀█  █  █ █  █ ",
		"▀ ▀ ▀▀▀ ▀ ▀  ▀ ",
	}
	got := Banner("MINT")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Banner(\"MINT\") =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// decode turns banner lines back into rows of pixels ('X' lit, '.' not).
func decodeBanner(t *testing.T, lines []string) []string {
	t.Helper()
	if len(lines) != BannerHeight {
		t.Fatalf("got %d lines, want %d", len(lines), BannerHeight)
	}
	rows := make([]string, 2*BannerHeight)
	for l, line := range lines {
		var top, bottom strings.Builder
		for _, r := range line {
			switch r {
			case '█':
				top.WriteByte('X')
				bottom.WriteByte('X')
			case '▀':
				top.WriteByte('X')
				bottom.WriteByte('.')
			case '▄':
				top.WriteByte('.')
				bottom.WriteByte('X')
			case ' ':
				top.WriteByte('.')
				bottom.WriteByte('.')
			default:
				t.Fatalf("line %d has unexpected character %q", l, r)
			}
		}
		rows[2*l], rows[2*l+1] = top.String(), bottom.String()
	}
	return rows
}

// Every glyph survives the trip into half-blocks and back, in every position.
func TestBannerRoundTrip(t *testing.T) {
	var all []rune
	for r := range bannerGlyphs {
		all = append(all, r)
	}
	text := string(all)
	rows := decodeBanner(t, Banner(text))
	if len(rows[0]) != BannerWidth(text) {
		t.Fatalf("decoded width %d, want %d", len(rows[0]), BannerWidth(text))
	}
	for i, r := range all {
		glyph := strings.Split(bannerGlyphs[r], "|")
		for y := 0; y < 5; y++ {
			got := rows[y][4*i : 4*i+3]
			if got != glyph[y] {
				t.Errorf("glyph %q row %d = %q, want %q", r, y, got, glyph[y])
			}
		}
		// the sixth pixel row only exists to pair up the fifth
		if rows[5][4*i:4*i+3] != "..." {
			t.Errorf("glyph %q: padding row is not blank", r)
		}
		// the column after each glyph is the gap
		if i < len(all)-1 {
			for y := 0; y < 6; y++ {
				if rows[y][4*i+3] != '.' {
					t.Errorf("glyph %q: gap column is lit on row %d", r, y)
				}
			}
		}
	}
}

func TestBannerGlyphTable(t *testing.T) {
	for r := 'A'; r <= 'Z'; r++ {
		if _, ok := bannerGlyphs[r]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
	for r := '0'; r <= '9'; r++ {
		if _, ok := bannerGlyphs[r]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
	for _, r := range " .,-_:!?'/+" {
		if _, ok := bannerGlyphs[r]; !ok {
			t.Errorf("no glyph for %q", r)
		}
	}
	for r, g := range bannerGlyphs {
		rows := strings.Split(g, "|")
		if len(rows) != 5 {
			t.Errorf("glyph %q has %d rows, want 5", r, len(rows))
			continue
		}
		for i, row := range rows {
			if len(row) != 3 || strings.Trim(row, ".X") != "" {
				t.Errorf("glyph %q row %d = %q, want 3 cells of . or X", r, i, row)
			}
		}
	}
	// Letters that must not look alike once packed into half-blocks.
	seen := map[string]rune{}
	for r := 'A'; r <= 'Z'; r++ {
		g := bannerGlyphs[r]
		if other, dup := seen[g]; dup {
			t.Errorf("%q and %q have the same glyph", other, r)
		}
		seen[g] = r
	}
}

func TestBannerSize(t *testing.T) {
	for _, text := range []string{"A", "MINT", "Hello, World", "a b"} {
		lines := Banner(text)
		if len(lines) != BannerHeight {
			t.Fatalf("Banner(%q): %d lines, want %d", text, len(lines), BannerHeight)
		}
		for i, l := range lines {
			if n := utf8.RuneCountInString(l); n != BannerWidth(text) {
				t.Errorf("Banner(%q) line %d is %d cells, want BannerWidth = %d", text, i, n, BannerWidth(text))
			}
			if VLen(l) != BannerWidth(text) {
				t.Errorf("Banner(%q) line %d: VLen = %d, want %d", text, i, VLen(l), BannerWidth(text))
			}
		}
	}
	if got := BannerWidth("MINT"); got != 15 {
		t.Errorf("BannerWidth(MINT) = %d, want 15", got)
	}
	if got := BannerWidth("é"); got != 3 {
		t.Errorf("BannerWidth(é) = %d, want 3 (one character, not its bytes)", got)
	}
	if Banner("") != nil || BannerWidth("") != 0 {
		t.Error("empty text should give no banner and no width")
	}
}

func TestBannerCaseAndUnknown(t *testing.T) {
	if a, b := strings.Join(Banner("mint 2026"), "\n"), strings.Join(Banner("MINT 2026"), "\n"); a != b {
		t.Errorf("lower case differs from upper case:\n%s\n%s", a, b)
	}
	q := strings.Join(Banner("?"), "\n")
	for _, s := range []string{"é", "\x1b", "\n", "◈", "🙂"} {
		if got := strings.Join(Banner(s), "\n"); got != q {
			t.Errorf("Banner(%q) should draw as ?, got\n%s", s, got)
		}
	}
	// Nothing in a banner can be anything but the three half-blocks and space,
	// whatever the text holds: it never carries a terminal escape.
	for _, l := range Banner("\x1b[31m<script>\x07") {
		if strings.Trim(l, "█▀▄ ") != "" {
			t.Errorf("banner line holds other characters: %q", l)
		}
	}
}

func TestBannerFits(t *testing.T) {
	cases := []struct {
		name      string
		h, w      int
		text      string
		noteRows  int
		asciiOnly bool
		want      bool
	}{
		{"standard terminal", 24, 80, "MINT", 0, false, true},
		{"standard terminal with a two-line note", 24, 80, "MINT", 2, false, true},
		{"no text", 24, 80, "", 0, false, false},
		{"ASCII-only terminal", 24, 80, "MINT", 0, true, false},
		{"exactly as wide as the frame allows", 24, 21, "MINT", 0, false, true},
		{"one column too narrow", 24, 20, "MINT", 0, false, false},
		{"title wider than the terminal", 24, 40, "A VERY LONG APPLICATION TITLE", 0, false, false},
		{"shortest terminal that keeps two items", 15, 80, "MINT", 0, false, true},
		{"one row too short", 14, 80, "MINT", 0, false, false},
		{"a note takes a row", 15, 80, "MINT", 1, false, false},
		{"a note and enough height", 16, 80, "MINT", 1, false, true},
	}
	for _, c := range cases {
		if got := bannerFits(c.h, c.w, c.text, c.noteRows, c.asciiOnly); got != c.want {
			t.Errorf("%s: bannerFits(h=%d, w=%d, %q, note=%d, ascii=%v) = %v, want %v",
				c.name, c.h, c.w, c.text, c.noteRows, c.asciiOnly, got, c.want)
		}
	}
}

// With no banner the list starts where it did before banners existed: row 4,
// or row 5 or 6 under a one- or two-line note.
func TestHomeHeaderRowsWithoutBanner(t *testing.T) {
	for noteRows, wantTop := range map[int]int{0: 4, 1: 5, 2: 6} {
		lines, noteTop, contentTop := homeHeaderRows(24, 80, "", noteRows, false)
		if lines != nil || noteTop != 4 || contentTop != wantTop {
			t.Errorf("note rows %d: got (%v, %d, %d), want (nil, 4, %d)", noteRows, lines, noteTop, contentTop, wantTop)
		}
	}
	// A banner that does not fit changes nothing either.
	for _, c := range []struct {
		h, w  int
		ascii bool
	}{{24, 80, true}, {24, 20, false}, {12, 80, false}} {
		lines, noteTop, contentTop := homeHeaderRows(c.h, c.w, "MINT", 1, c.ascii)
		if lines != nil || noteTop != 4 || contentTop != 5 {
			t.Errorf("banner that does not fit (%dx%d ascii=%v): got (%v, %d, %d)", c.w, c.h, c.ascii, lines, noteTop, contentTop)
		}
	}
}

func TestHomeHeaderRowsWithBanner(t *testing.T) {
	// The banner takes rows 4–6 and row 7 is blank, so the note starts on row 8.
	lines, noteTop, contentTop := homeHeaderRows(24, 80, "MINT", 0, false)
	if len(lines) != BannerHeight || noteTop != 8 || contentTop != 8 {
		t.Errorf("no note: got (%d lines, %d, %d), want (%d lines, 8, 8)", len(lines), noteTop, contentTop, BannerHeight)
	}
	lines, noteTop, contentTop = homeHeaderRows(24, 80, "MINT", 2, false)
	if len(lines) != BannerHeight || noteTop != 8 || contentTop != 10 {
		t.Errorf("two-line note: got (%d lines, %d, %d), want (%d lines, 8, 10)", len(lines), noteTop, contentTop, BannerHeight)
	}
	if strings.Join(lines, "\n") != strings.Join(Banner("MINT"), "\n") {
		t.Error("homeHeaderRows should return exactly Banner's lines")
	}
}
