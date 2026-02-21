//go:build !android

package game

import (
	"math"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Car texture variants stored on the renderer.
type CarTextures struct {
	base     uint32
	variants []uint32
}

func makeCarTexture(r *Rand) uint32 {
	const s = 8
	pix := make([]uint8, s*s*4)

	body := RGB{R: uint8(180 + r.Range(-40, 40)), G: uint8(80 + r.Range(-20, 20)), B: uint8(70 + r.Range(-20, 20))}
	window := RGB{R: 140, G: 140, B: 140}
	roof := body.Mul(180)

	set := func(x, y int, col RGB) {
		i := (y*s + x) * 4
		pix[i+0] = col.R
		pix[i+1] = col.G
		pix[i+2] = col.B
		pix[i+3] = 255
	}

	// Vertical bands: front, window, roof, trunk.
	for y := 0; y < s; y++ {
		var col RGB
		switch y / 2 {
		case 0:
			col = body
		case 1:
			col = window
		case 2:
			col = roof
		default:
			col = body
		}
		for x := 0; x < s; x++ {
			set(x, y, col)
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, s, s, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	return tex
}

func makeCarTextureVariant(seed uint64) uint32 {
	const s = 8
	pix := make([]uint8, s*s*4)

	r := NewRand(seed)
	body := RGB{R: uint8(140 + r.Range(0, 80)), G: uint8(60 + r.Range(0, 80)), B: uint8(50 + r.Range(0, 80))}
	window := RGB{R: 130, G: 135, B: 140}
	roof := body.Mul(180)

	set := func(x, y int, col RGB) {
		i := (y*s + x) * 4
		pix[i+0] = col.R
		pix[i+1] = col.G
		pix[i+2] = col.B
		pix[i+3] = 255
	}

	// Archetypes with varying band heights.
	archetypes := [][]int{
		{2, 2, 2, 2}, {2, 2, 1, 3}, {1, 2, 3, 2},
		{1, 1, 4, 2}, {2, 1, 3, 2}, {1, 3, 2, 2},
	}
	bands := archetypes[r.Intn(len(archetypes))]

	y := 0
	for bi := 0; bi < len(bands) && y < s; bi++ {
		var col RGB
		switch bi {
		case 0:
			col = body
		case 1:
			col = window
		case 2:
			col = roof
		default:
			col = body
		}
		for row := 0; row < bands[bi] && y < s; row++ {
			for x := 0; x < s; x++ {
				set(x, y, col)
			}
			y++
		}
	}
	for ; y < s; y++ {
		for x := 0; x < s; x++ {
			set(x, y, body)
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, s, s, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	return tex
}

// makeCopCarTexture creates the police car texture:
// white front bumper/hood, blue-tinted windows, blue roof, white rear trunk/bumper.
func makeCopCarTexture() uint32 {
	const s = 8
	pix := make([]uint8, s*s*4)

	white := RGB{R: 230, G: 230, B: 235}
	blue := RGB{R: 60, G: 100, B: 220}
	window := RGB{R: 40, G: 60, B: 140} // dark blue-tinted glass
	roof := RGB{R: 50, G: 85, B: 200}   // slightly darker roof for light-bar contrast

	bands := []struct {
		h   int
		col RGB
	}{
		{1, white},  // front bumper
		{1, white},  // hood
		{1, window}, // windshield
		{1, window}, // front windows
		{2, roof},   // roof (light bar sits here)
		{1, blue},   // trunk
		{1, white},  // rear bumper
	}

	set := func(x, y int, col RGB) {
		i := (y*s + x) * 4
		pix[i+0] = col.R
		pix[i+1] = col.G
		pix[i+2] = col.B
		pix[i+3] = 255
	}

	y := 0
	for _, b := range bands {
		for row := 0; row < b.h && y < s; row++ {
			for x := 0; x < s; x++ {
				set(x, y, b.col)
			}
			y++
		}
	}

	var tex uint32
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(gl.TEXTURE_2D, 0, gl.RGBA8, s, s, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(pix))
	return tex
}

// InitCarTextures creates car textures on the renderer.
func (r *Renderer) InitCarTextures() {
	rng := NewRand(0xC0FFEE)
	r.carTexBase = makeCarTexture(rng)
	r.carTexVariants = make([]uint32, 8)
	for i := range r.carTexVariants {
		r.carTexVariants[i] = makeCarTextureVariant(rng.NextU64())
	}
	r.copCarTex = makeCopCarTexture()
}

// DrawNPCCars renders NPC cars using the chunk program (rotated textured quads).
func (rend *Renderer) DrawNPCCars(ts *TrafficSystem, cam Camera, fbW, fbH int) {
	if ts == nil || len(ts.Cars) == 0 {
		return
	}

	gl.UseProgram(rend.chunkProg)
	gl.BindVertexArray(rend.chunkVAO)
	gl.Uniform2f(rend.uCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(rend.uZoom, float32(cam.Zoom))
	gl.Uniform2f(rend.uResolution, float32(fbW), float32(fbH))

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	for i, c := range ts.Cars {
		if !c.Alive {
			continue
		}
		tex := rend.carTexBase
		if len(rend.carTexVariants) > 0 {
			tex = rend.carTexVariants[i%len(rend.carTexVariants)]
		}

		gl.Uniform2f(rend.uChunkSize, float32(CarSize*CarVisualAspect), float32(CarSize))
		gl.Uniform2f(rend.uChunkOrigin, float32(c.X-(CarSize*0.5*CarVisualAspect)), float32(c.Y-CarSize*0.5))
		gl.Uniform1f(rend.uRotation, float32(c.Heading+math.Pi*0.5))

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, tex)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
	}

	gl.Disable(gl.BLEND)

	// Restore chunk defaults.
	gl.Uniform2f(rend.uChunkSize, float32(ChunkSize), float32(ChunkSize))
	gl.Uniform1f(rend.uRotation, 0)
}

// DrawCopCars renders police cars as textured quads using the police car texture.
func (rend *Renderer) DrawCopCars(cs *CopSystem, cam Camera, fbW, fbH int) {
	if cs == nil || len(cs.Cars) == 0 {
		return
	}

	gl.UseProgram(rend.chunkProg)
	gl.BindVertexArray(rend.chunkVAO)
	gl.Uniform2f(rend.uCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(rend.uZoom, float32(cam.Zoom))
	gl.Uniform2f(rend.uResolution, float32(fbW), float32(fbH))

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	for _, c := range cs.Cars {
		if !c.Alive {
			continue
		}
		gl.Uniform2f(rend.uChunkSize, float32(CarSize*CarVisualAspect), float32(CarSize))
		gl.Uniform2f(rend.uChunkOrigin, float32(c.X-(CarSize*0.5*CarVisualAspect)), float32(c.Y-CarSize*0.5))
		gl.Uniform1f(rend.uRotation, float32(c.Heading+math.Pi*0.5))

		gl.ActiveTexture(gl.TEXTURE0)
		gl.BindTexture(gl.TEXTURE_2D, rend.copCarTex)
		gl.DrawArrays(gl.TRIANGLES, 0, 6)
	}

	gl.Disable(gl.BLEND)

	gl.Uniform2f(rend.uChunkSize, float32(ChunkSize), float32(ChunkSize))
	gl.Uniform1f(rend.uRotation, 0)
}

// CarShadowSprites returns soft shadow sprites for NPC and cop cars.
// Drawn before the car textures so the shadow appears underneath the car body.
// Each car gets three overlapping circles along its forward axis, offset south-east.
// Pass a reusable buf (reset to [:0] internally) to avoid per-frame allocations.
func CarShadowSprites(ts *TrafficSystem, cs *CopSystem, buf []float32) []float32 {
	buf = buf[:0]

	addShadow := func(x, y, heading float64) {
		fwdX := math.Cos(heading)
		fwdY := math.Sin(heading)
		ox := x + 0.6 // south-east offset
		oy := y + 1.0
		sz := float32(CarSize * 0.95)
		for _, t := range [3]float64{-1.6, 0, 1.6} {
			buf = append(buf,
				float32(ox+fwdX*t), float32(oy+fwdY*t),
				sz, 0, 0, 0, 0.22, 0)
		}
	}

	for _, c := range ts.Cars {
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
	for _, c := range cs.Cars {
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
	return buf
}

// CarHeadlightSprites returns glow sprite data for car headlights.
// brightness: 0=day (off), 1=midnight (full). RGB is pre-multiplied for additive blending.
// Pass a reusable buf (reset to [:0] internally) to avoid per-frame allocations.
func CarHeadlightSprites(ts *TrafficSystem, brightness float32, buf []float32) []float32 {
	buf = buf[:0]
	if ts == nil || brightness <= 0.01 {
		return buf
	}
	for i := range ts.Cars {
		c := &ts.Cars[i]
		if !c.Alive {
			continue
		}
		offset := float64(c.Size) * 0.5
		perpX := float32(-math.Sin(c.Heading) * 0.9)
		perpY := float32(math.Cos(c.Heading) * 0.9)
		frontX := float32(c.X + math.Cos(c.Heading)*offset)
		frontY := float32(c.Y + math.Sin(c.Heading)*offset)
		rearX := float32(c.X - math.Cos(c.Heading)*offset)
		rearY := float32(c.Y - math.Sin(c.Heading)*offset)
		sz := 2.5 * brightness

		// Front headlights: warm white, pre-multiplied.
		hw := 1.0 * brightness
		buf = append(buf, frontX+perpX, frontY+perpY, sz, hw, 0.95*brightness, 0.65*brightness, 1, 0)
		buf = append(buf, frontX-perpX, frontY-perpY, sz, hw, 0.95*brightness, 0.65*brightness, 1, 0)
		// Rear taillights: red, pre-multiplied.
		rw := 0.7 * brightness
		buf = append(buf, rearX+perpX, rearY+perpY, sz*0.7, rw, 0.03*brightness, 0.03*brightness, 1, 0)
		buf = append(buf, rearX-perpX, rearY-perpY, sz*0.7, rw, 0.03*brightness, 0.03*brightness, 1, 0)
	}
	return buf
}

// DrawPedestrians renders pedestrians as point sprites.
// Reuses rend.pedBuf across frames to avoid per-frame heap allocations.
func (rend *Renderer) DrawPedestrians(peds *PedestrianSystem, cam Camera, fbW, fbH int, now float64) {
	if peds == nil || len(peds.P) == 0 {
		return
	}
	rend.pedBuf = peds.PedRenderData(rend.pedBuf, now)
	rend.DrawSprites(rend.pedBuf, cam, fbW, fbH, false)
	rend.RestoreChunkProgram()
}
