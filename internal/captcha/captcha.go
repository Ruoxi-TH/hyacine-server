package captcha

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"time"
)

type Generator struct {
	width  int
	height int
}

func NewGenerator() *Generator {
	return &Generator{
		width:  200,
		height: 80,
	}
}

func (g *Generator) Generate(code string) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))
	bgColor := color.RGBA{245, 245, 245, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 100; i++ {
		x := r.Intn(g.width)
		y := r.Intn(g.height)
		img.Set(x, y, color.RGBA{uint8(180 + r.Intn(75)), uint8(180 + r.Intn(75)), uint8(180 + r.Intn(75)), 255})
	}
	for i := 0; i < 6; i++ {
		drawLine(img, r.Intn(g.width), r.Intn(g.height), r.Intn(g.width), r.Intn(g.height), color.RGBA{uint8(100 + r.Intn(100)), uint8(100 + r.Intn(100)), uint8(100 + r.Intn(100)), 255})
	}
	charWidth := g.width / 5
	for i, c := range code {
		x := 15 + i*charWidth
		y := 15 + r.Intn(15)
		drawChar(img, x, y, string(c), color.RGBA{uint8(30 + r.Intn(80)), uint8(30 + r.Intn(80)), uint8(30 + r.Intn(80)), 255}, 2)
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx, dy := abs(x2-x1), abs(y2-y1)
	sx, sy := 1, 1
	if x1 >= x2 { sx = -1 }
	if y1 >= y2 { sy = -1 }
	err := dx - dy
	for {
		for ddx := -1; ddx <= 1; ddx++ {
			for ddy := -1; ddy <= 1; ddy++ {
				px, py := x1+ddx, y1+ddy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, c)
				}
			}
		}
		if x1 == x2 && y1 == y2 { break }
		e2 := err * 2
		if e2 > -dy { err -= dy; x1 += sx }
		if e2 < dx { err += dx; y1 += sy }
	}
}

func drawChar(img *image.RGBA, x, y int, char string, c color.Color, scale int) {
	font := map[string][]string{
		"A": {"  ##  ", " #  # ", "######", "#    #", "#    #"},
		"B": {"##### ", "#    #", "##### ", "#    #", "##### "},
		"C": {" #####", "#     ", "#     ", "#     ", " #####"},
		"D": {"##### ", "#    #", "#    #", "#    #", "##### "},
		"E": {"######", "#     ", "####  ", "#     ", "######"},
		"F": {"######", "#     ", "####  ", "#     ", "#     "},
		"G": {" #####", "#     ", "#  ###", "#    #", " #####"},
		"H": {"#    #", "#    #", "######", "#    #", "#    #"},
		"J": {"    ##", "     #", "     #", "#    #", " #####"},
		"K": {"#   # ", "# #   ", "##    ", "# #   ", "#   # "},
		"L": {"#     ", "#     ", "#     ", "#     ", "######"},
		"M": {"#    #", "##  ##", "# ## #", "#    #", "#    #"},
		"N": {"#    #", "##   #", "# #  #", "#  # #", "#   ##"},
		"P": {"######", "#    #", "######", "#     ", "#     "},
		"Q": {" #####", "#    #", "#  # #", "#   ##", " #### "},
		"R": {"##### ", "#    #", "##### ", "#  #  ", "#   # "},
		"S": {" #####", "#     ", " #####", "     #", "##### "},
		"T": {"######", "  ##  ", "  ##  ", "  ##  ", "  ##  "},
		"U": {"#    #", "#    #", "#    #", "#    #", " #####"},
		"V": {"#    #", "#    #", " #  # ", " #  # ", "  ##  "},
		"W": {"#    #", "#  # #", "# ## #", "##  ##", "#    #"},
		"X": {"#    #", " #  # ", "  ##  ", " #  # ", "#    #"},
		"Y": {"#    #", " #  # ", "  ##  ", "  ##  ", "  ##  "},
		"Z": {"######", "   ## ", "  ##  ", " ##   ", "######"},
		"2": {" #####", "#    #", "   ## ", " #    ", "######"},
		"3": {"##### ", "    # ", " #### ", "    # ", "##### "},
		"4": {"#    #", "#    #", "######", "     #", "     #"},
		"5": {"######", "#     ", "##### ", "     #", "##### "},
		"6": {" #####", "#     ", "##### ", "#    #", " #####"},
		"7": {"######", "    # ", "   #  ", "  #   ", " #    "},
		"8": {" #####", "#    #", " #####", "#    #", " #####"},
		"9": {" #####", "#    #", " ######", "     #", " #####"},
	}
	pattern, ok := font[char]
	if !ok { return }
	for dy, row := range pattern {
		for dx, pixel := range row {
			if pixel == '#' {
				for sx := 0; sx < scale; sx++ {
					for sy := 0; sy < scale; sy++ {
						px := x + dx*scale + sx
						py := y + dy*scale + sy
						if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
							img.Set(px, py, c)
						}
					}
				}
			}
		}
	}
}

func abs(x int) int { if x < 0 { return -x }; return x }
