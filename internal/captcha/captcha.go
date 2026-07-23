package captcha

import (
	"bytes"
	"encoding/base64"
	"fmt"
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
		width:  120,
		height: 40,
	}
}

func (g *Generator) Generate(code string) (string, error) {
	img := image.NewRGBA(image.Rect(0, 0, g.width, g.height))
	
	bgColor := color.RGBA{240, 240, 240, 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)
	
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 50; i++ {
		x := r.Intn(g.width)
		y := r.Intn(g.height)
		noiseColor := color.RGBA{
			uint8(r.Intn(200)),
			uint8(r.Intn(200)),
			uint8(r.Intn(200)),
			255,
		}
		img.Set(x, y, noiseColor)
	}
	
	for i := 0; i < 4; i++ {
		x1 := r.Intn(g.width)
		y1 := r.Intn(g.height)
		x2 := r.Intn(g.width)
		y2 := r.Intn(g.height)
		lineColor := color.RGBA{
			uint8(100 + r.Intn(100)),
			uint8(100 + r.Intn(100)),
			uint8(100 + r.Intn(100)),
			255,
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	
	charWidth := g.width / 5
	for i, c := range code {
		x := 10 + i*charWidth
		y := 10 + r.Intn(10)
		charColor := color.RGBA{
			uint8(50 + r.Intn(100)),
			uint8(50 + r.Intn(100)),
			uint8(50 + r.Intn(100)),
			255,
		}
		drawChar(img, x, y, string(c), charColor)
	}
	
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", err
	}
	
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 >= x2 {
		sx = -1
	}
	if y1 >= y2 {
		sy = -1
	}
	err := dx - dy
	
	for {
		img.Set(x1, y1, c)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func drawChar(img *image.RGBA, x, y int, char string, c color.Color) {
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
	if !ok {
		return
	}
	
	for dy, row := range pattern {
		for dx, pixel := range row {
			if pixel == '#' {
				img.Set(x+dx, y+dy, c)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func FormatFrom(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, email)
}