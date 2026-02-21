package game

// RGB is an 8-bit per channel colour.
type RGB struct {
	R, G, B uint8
}

func (c RGB) Mul(k uint8) RGB {
	return RGB{
		R: uint8((uint16(c.R) * uint16(k)) / 255),
		G: uint8((uint16(c.G) * uint16(k)) / 255),
		B: uint8((uint16(c.B) * uint16(k)) / 255),
	}
}

func (c RGB) Add(dr, dg, db int) RGB {
	r := int(c.R) + dr
	g := int(c.G) + dg
	b := int(c.B) + db
	if r < 0 {
		r = 0
	} else if r > 255 {
		r = 255
	}
	if g < 0 {
		g = 0
	} else if g > 255 {
		g = 255
	}
	if b < 0 {
		b = 0
	} else if b > 255 {
		b = 255
	}
	return RGB{R: uint8(r), G: uint8(g), B: uint8(b)}
}

var Palette = struct {
	Road         RGB
	Sidewalk     RGB
	Lot          RGB
	BuildingA    RGB
	BuildingB    RGB
	BuildingC    RGB
	BuildingDark RGB
	Grass        RGB
	GrassPatch   RGB
	GrassTorn    RGB
	TreeBase     RGB
	TreeMid      RGB
	TreeTop      RGB
	Rubble       RGB
	Border       RGB
	Smoke        RGB
	Glow         RGB
	FireHot      RGB
	FireMid      RGB
	FireCool     RGB
}{
	Road:         RGB{R: 60, G: 66, B: 79},
	Sidewalk:     RGB{R: 214, G: 190, B: 153},
	Lot:          RGB{R: 216, G: 210, B: 191},
	BuildingA:    RGB{R: 153, G: 144, B: 133},
	BuildingB:    RGB{R: 104, G: 108, B: 112},
	BuildingC:    RGB{R: 195, G: 174, B: 142},
	BuildingDark: RGB{R: 86, G: 89, B: 88},
	Grass:        RGB{R: 140, G: 136, B: 91},
	GrassPatch:   RGB{R: 122, G: 120, B: 78},
	GrassTorn:    RGB{R: 160, G: 150, B: 92},
	TreeBase:     RGB{R: 70, G: 95, B: 50},
	TreeMid:      RGB{R: 90, G: 120, B: 65},
	TreeTop:      RGB{R: 120, G: 150, B: 85},
	Rubble:       RGB{R: 104, G: 108, B: 112},
	Border:       RGB{R: 0, G: 0, B: 0},
	Smoke:        RGB{R: 120, G: 120, B: 125},
	Glow:         RGB{R: 255, G: 200, B: 90},
	FireHot:      RGB{R: 255, G: 210, B: 110},
	FireMid:      RGB{R: 255, G: 150, B: 70},
	FireCool:     RGB{R: 190, G: 70, B: 45},
}
