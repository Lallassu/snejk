//go:build android

package game

// streetlightCache avoids rebuilding the streetlight sprite buffer every frame.
var streetlightCache struct {
	brightness float32
	buf        []float32
}

// streetlightSprites returns radial glow sprites for road intersection lights.
func streetlightSprites(brightness float32) []float32 {
	q := float32(int(brightness*200)) / 200.0
	if streetlightCache.buf != nil && streetlightCache.brightness == q {
		return streetlightCache.buf
	}
	buf := make([]float32, 0, 128)
	for y := 0; y+RoadWidth < WorldHeight; y += Pattern {
		for x := 0; x+RoadWidth < WorldWidth; x += Pattern {
			fx := float32(x + RoadWidth)
			fy := float32(y + RoadWidth)
			buf = append(buf, fx, fy, 10.0, 0.5*brightness, 0.42*brightness, 0.15*brightness, 1, 0)
			buf = append(buf, fx, fy, 2.0, 1.0*brightness, 1.0*brightness, 0.7*brightness, 1, 0)
		}
	}
	streetlightCache.brightness = q
	streetlightCache.buf = buf
	return buf
}
