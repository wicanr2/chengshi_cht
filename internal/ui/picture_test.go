package ui

import (
	"image/color"
	"math"
	"testing"
)

func TestPictureOverlayTextContrast(t *testing.T) {
	bg := color.RGBAModel.Convert(pictureBG).(color.RGBA)
	for name, fg := range map[string]color.RGBA{
		"正文": pictureText,
		"提示": pictureHint,
		"邊框": pictureBorder,
	} {
		if ratio := contrastRatio(fg, bg); ratio < 4.5 {
			t.Errorf("%s對比度 %.2f，小於 4.5", name, ratio)
		}
	}
}

func TestScenarioBriefGeometryMatchesDOS(t *testing.T) {
	if got := [4]int{briefX, briefY, briefW, briefH}; got != [4]int{168, 85, 304, 166} {
		t.Fatalf("劇本簡介矩形 = %v，DOS 原版量測 = [168 85 304 166]", got)
	}
	if briefButtonY != 222 || briefButtonH != 20 {
		t.Fatalf("劇本簡介按鈕 y/h = %d/%d，DOS 原版量測 = 222/20", briefButtonY, briefButtonH)
	}
}

func contrastRatio(a, b color.RGBA) float64 {
	la, lb := relativeLuminance(a), relativeLuminance(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func relativeLuminance(c color.RGBA) float64 {
	linear := func(v uint8) float64 {
		x := float64(v) / 255
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(c.R) + 0.7152*linear(c.G) + 0.0722*linear(c.B)
}
