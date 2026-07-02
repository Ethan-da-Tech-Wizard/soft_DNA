//go:build ignore

// One-off generator for icon.png — a simple, flat geometric mark (no font
// rendering needed) so codelens-gui has something other than Fyne's default
// logo in the taskbar/dock. Regenerate with:
//
//	cd internal/gui/assets && go run gen_icon.go
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

const size = 128

// roundedRectAlpha returns 0..1 coverage for point (x,y) inside a rounded
// rectangle spanning [0,0]-[w,h] with corner radius r, antialiased over one
// pixel at the boundary.
func roundedRectAlpha(x, y, w, h, r float64) float64 {
	cx := x
	if cx > w-r {
		cx = w - r
	} else if cx < r {
		cx = r
	}
	cy := y
	if cy > h-r {
		cy = h - r
	} else if cy < r {
		cy = r
	}
	dx := x - cx
	dy := y - cy
	inCornerBox := (x < r || x > w-r) && (y < r || y > h-r)
	if !inCornerBox {
		if x >= 0 && x <= w && y >= 0 && y <= h {
			return 1
		}
		return 0
	}
	dist := dx*dx + dy*dy
	rr := r * r
	if dist <= rr {
		return 1
	}
	edge := (dist - rr) / (2*r + 1)
	if edge > 1 {
		return 0
	}
	return 1 - edge
}

func lerp(a, b color.NRGBA, t float64) color.NRGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: l(a.A, b.A)}
}

func main() {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))

	outer := color.NRGBA{R: 0x14, G: 0x1a, B: 0x2e, A: 0xff} // dark navy background
	inner := color.NRGBA{R: 0x2f, G: 0x6f, B: 0xed, A: 0xff} // accent blue (matches Fyne's primary)
	transparent := color.NRGBA{}

	const outerMargin, outerR = 4.0, 26.0
	const innerMargin, innerR = 30.0, 14.0

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			ox, oy := float64(x)-outerMargin, float64(y)-outerMargin
			ow := float64(size) - 2*outerMargin
			oa := roundedRectAlpha(ox, oy, ow, ow, outerR)

			ix, iy := float64(x)-innerMargin, float64(y)-innerMargin
			iw := float64(size) - 2*innerMargin
			ia := roundedRectAlpha(ix, iy, iw, iw, innerR)

			c := lerp(transparent, outer, oa)
			c = lerp(c, inner, ia)
			img.SetNRGBA(x, y, c)
		}
	}

	f, err := os.Create("icon.png")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}
