//go:build !android

package game

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/go-gl/gl/v4.1-core/gl"
)

const (
	DayCyclePeriod = 90.0 // seconds of game time per full day/night cycle
	SunAmbientMin  = 0.38 // midnight ambient floor (raised from 0.30)
	SunAmbientMax  = 1.00 // noon ambient
	SunNightStart  = 0.65 // ambient threshold where night lighting kicks in
)

// SunCycleLight computes ambient light level and color tint from game time.
// Returns ambient (SunAmbientMin..SunAmbientMax), and tint RGB multipliers.
func SunCycleLight(gameTime float64) (ambient, tintR, tintG, tintB float32) {
	phase := math.Mod(gameTime, DayCyclePeriod) / DayCyclePeriod // 0..1
	sunHeight := math.Sin(phase * 2 * math.Pi)                   // -1 (midnight) to 1 (noon)

	// Ambient: SunAmbientMin (midnight) to SunAmbientMax (noon).
	mid := float64(SunAmbientMin+SunAmbientMax) * 0.5
	amp := float64(SunAmbientMax-SunAmbientMin) * 0.5
	ambient = float32(mid + amp*sunHeight)

	// Warm orange tint near horizon (sunHeight near 0 = sunset/sunrise).
	horizonFactor := 1.0 - math.Abs(sunHeight)
	warmth := horizonFactor * horizonFactor * 0.35
	tintR = float32(1.0 + warmth*0.4)
	tintG = float32(1.0 - warmth*0.15)
	tintB = float32(1.0 - warmth*0.5)

	// Slight blue tint at night.
	if sunHeight < -0.3 {
		nightFactor := float32((-sunHeight - 0.3) / 0.7)
		// Slightly gentler night tint to keep visibility higher.
		tintR -= nightFactor * 0.07
		tintG -= nightFactor * 0.035
		tintB += nightFactor * 0.10
	}

	return
}

// NightIntensityFromAmbient maps ambient light to a 0..1 night factor.
// 0 at/above SunNightStart, 1 at SunAmbientMin.
func NightIntensityFromAmbient(ambient float32) float32 {
	denom := float64(SunNightStart - SunAmbientMin)
	if denom <= 0 {
		return 0
	}
	return float32(clampF((float64(SunNightStart)-float64(ambient))/denom, 0, 1))
}

// SunCycleShadow computes a continuous sun angle and shadow slope from game time.
// The angle rotates smoothly so shadows sweep around as the sun crosses the sky.
func SunCycleShadow(gameTime float64) (angle, slope float64) {
	phase := math.Mod(gameTime, DayCyclePeriod) / DayCyclePeriod
	sunHeight := math.Sin(phase * 2 * math.Pi)

	// Sun angle rotates clockwise: dawn=east(0), noon=north(-π/2), dusk=west(-π), midnight=south(-3π/2).
	angle = -phase * 2 * math.Pi

	// Shadow slope: higher = shorter shadows.
	// Daytime: 1.0 (horizon) to 3.0 (noon). Night: 1.0 (long shadows).
	if sunHeight > 0 {
		slope = 1.0 + sunHeight*2.0
	} else {
		slope = 1.0
	}
	return
}

// glOffset converts a byte offset to unsafe.Pointer for OpenGL VBO offset params.
func glOffset(n int) unsafe.Pointer { return unsafe.Pointer(uintptr(n)) }

type Renderer struct {
	// Chunk program.
	chunkProg uint32
	chunkVAO  uint32
	chunkVBO  uint32

	uChunkOrigin  int32
	uChunkSize    int32
	uRotation     int32
	uCamera       int32
	uZoom         int32
	uResolution   int32
	uTex          int32
	chunkUAmbient int32
	chunkUSunTint int32

	// Particle/sprite program.
	spriteProg uint32
	spriteVAO  uint32
	spriteVBO  uint32

	spUCamera     int32
	spUZoom       int32
	spUResolution int32
	spUAmbient    int32
	spUSunTint    int32

	// Glow (radial light) program — uses spriteVAO, additive blend only.
	glowProg        uint32
	glowUCamera     int32
	glowUZoom       int32
	glowUResolution int32

	// NPC (car) program.
	npcProg        uint32
	npcUCamera     int32
	npcUZoom       int32
	npcUResolution int32
	npcUCarTex     int32
	npcUCarAspect  int32

	// Bonus box program.
	bonusProg        uint32
	bonusUCamera     int32
	bonusUZoom       int32
	bonusUResolution int32
	bonusUAmbient    int32
	bonusUSunTint    int32

	// Car textures.
	carTexBase     uint32
	carTexVariants []uint32
	copCarTex      uint32

	// Font/text rendering.
	fontTex      uint32
	textProg     uint32
	textVAO      uint32
	textVBO      uint32
	textURes     int32
	textUFontTex int32
	textBuf      []float32

	// Reusable render buffers to avoid per-frame heap allocations.
	pedBuf []float32
}

func NewRenderer() (*Renderer, error) {
	chunkProg, err := linkProgram(chunkVertSrc, chunkFragSrc)
	if err != nil {
		return nil, fmt.Errorf("chunk program: %w", err)
	}
	spriteProg, err := linkProgram(particleVertSrc, particleFragSrc)
	if err != nil {
		gl.DeleteProgram(chunkProg)
		return nil, fmt.Errorf("sprite program: %w", err)
	}
	glowProg, err := linkProgram(particleVertSrc, glowFragSrc)
	if err != nil {
		gl.DeleteProgram(chunkProg)
		gl.DeleteProgram(spriteProg)
		return nil, fmt.Errorf("glow program: %w", err)
	}
	npcProg, err := linkProgram(particleVertSrc, npcFragSrc)
	if err != nil {
		gl.DeleteProgram(chunkProg)
		gl.DeleteProgram(spriteProg)
		gl.DeleteProgram(glowProg)
		return nil, fmt.Errorf("npc program: %w", err)
	}
	bonusProg, err := linkProgram(particleVertSrc, bonusFragSrc)
	if err != nil {
		gl.DeleteProgram(chunkProg)
		gl.DeleteProgram(spriteProg)
		gl.DeleteProgram(glowProg)
		gl.DeleteProgram(npcProg)
		return nil, fmt.Errorf("bonus program: %w", err)
	}

	r := &Renderer{
		chunkProg:  chunkProg,
		spriteProg: spriteProg,
		glowProg:   glowProg,
		npcProg:    npcProg,
		bonusProg:  bonusProg,
	}

	// Chunk VAO/VBO: a unit quad (6 vertices, 2 triangles).
	var cVAO, cVBO uint32
	gl.GenVertexArrays(1, &cVAO)
	gl.GenBuffers(1, &cVBO)
	gl.BindVertexArray(cVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, cVBO)

	quadVerts := [12]float32{
		0, 0, 1, 0, 1, 1,
		0, 0, 1, 1, 0, 1,
	}
	gl.BufferData(gl.ARRAY_BUFFER, len(quadVerts)*4, gl.Ptr(&quadVerts[0]), gl.STATIC_DRAW)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, 2*4, glOffset(0))
	r.chunkVAO = cVAO
	r.chunkVBO = cVBO

	// Chunk uniforms.
	gl.UseProgram(chunkProg)
	r.uChunkOrigin = gl.GetUniformLocation(chunkProg, gl.Str("uChunkOrigin\x00"))
	r.uChunkSize = gl.GetUniformLocation(chunkProg, gl.Str("uChunkSize\x00"))
	r.uRotation = gl.GetUniformLocation(chunkProg, gl.Str("uRotation\x00"))
	r.uCamera = gl.GetUniformLocation(chunkProg, gl.Str("uCamera\x00"))
	r.uZoom = gl.GetUniformLocation(chunkProg, gl.Str("uZoom\x00"))
	r.uResolution = gl.GetUniformLocation(chunkProg, gl.Str("uResolution\x00"))
	r.uTex = gl.GetUniformLocation(chunkProg, gl.Str("uTex\x00"))
	gl.Uniform1i(r.uTex, 0)
	r.chunkUAmbient = gl.GetUniformLocation(chunkProg, gl.Str("uAmbient\x00"))
	r.chunkUSunTint = gl.GetUniformLocation(chunkProg, gl.Str("uSunTint\x00"))
	gl.Uniform1f(r.chunkUAmbient, 1.0)
	gl.Uniform3f(r.chunkUSunTint, 1.0, 1.0, 1.0)

	// Sprite VAO/VBO: streaming buffer for point sprites.
	// Each sprite: 8 floats (x, y, size, r, g, b, a, rotation).
	var sVAO, sVBO uint32
	gl.GenVertexArrays(1, &sVAO)
	gl.GenBuffers(1, &sVBO)
	gl.BindVertexArray(sVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, sVBO)

	stride := int32(8 * 4)
	gl.BufferData(gl.ARRAY_BUFFER, MaxParticleRender*int(stride), nil, gl.STREAM_DRAW)
	// aWorldPos (vec2)
	gl.EnableVertexAttribArray(0)
	gl.VertexAttribPointer(0, 2, gl.FLOAT, false, stride, glOffset(0))
	// aSize (float)
	gl.EnableVertexAttribArray(1)
	gl.VertexAttribPointer(1, 1, gl.FLOAT, false, stride, glOffset(2*4))
	// aColor (vec4)
	gl.EnableVertexAttribArray(2)
	gl.VertexAttribPointer(2, 4, gl.FLOAT, false, stride, glOffset(3*4))
	// aRotation (float)
	gl.EnableVertexAttribArray(3)
	gl.VertexAttribPointer(3, 1, gl.FLOAT, false, stride, glOffset(7*4))
	r.spriteVAO = sVAO
	r.spriteVBO = sVBO

	// Sprite uniforms.
	gl.UseProgram(spriteProg)
	r.spUCamera = gl.GetUniformLocation(spriteProg, gl.Str("uCamera\x00"))
	r.spUZoom = gl.GetUniformLocation(spriteProg, gl.Str("uZoom\x00"))
	r.spUResolution = gl.GetUniformLocation(spriteProg, gl.Str("uResolution\x00"))
	r.spUAmbient = gl.GetUniformLocation(spriteProg, gl.Str("uAmbient\x00"))
	r.spUSunTint = gl.GetUniformLocation(spriteProg, gl.Str("uSunTint\x00"))
	gl.Uniform1f(r.spUAmbient, 1.0)
	gl.Uniform3f(r.spUSunTint, 1.0, 1.0, 1.0)

	// Glow uniforms.
	gl.UseProgram(glowProg)
	r.glowUCamera = gl.GetUniformLocation(glowProg, gl.Str("uCamera\x00"))
	r.glowUZoom = gl.GetUniformLocation(glowProg, gl.Str("uZoom\x00"))
	r.glowUResolution = gl.GetUniformLocation(glowProg, gl.Str("uResolution\x00"))

	// NPC uniforms.
	gl.UseProgram(npcProg)
	r.npcUCamera = gl.GetUniformLocation(npcProg, gl.Str("uCamera\x00"))
	r.npcUZoom = gl.GetUniformLocation(npcProg, gl.Str("uZoom\x00"))
	r.npcUResolution = gl.GetUniformLocation(npcProg, gl.Str("uResolution\x00"))
	r.npcUCarTex = gl.GetUniformLocation(npcProg, gl.Str("uCarTex\x00"))
	gl.Uniform1i(r.npcUCarTex, 1)
	r.npcUCarAspect = gl.GetUniformLocation(npcProg, gl.Str("uCarAspect\x00"))

	// Bonus box uniforms.
	gl.UseProgram(bonusProg)
	r.bonusUCamera = gl.GetUniformLocation(bonusProg, gl.Str("uCamera\x00"))
	r.bonusUZoom = gl.GetUniformLocation(bonusProg, gl.Str("uZoom\x00"))
	r.bonusUResolution = gl.GetUniformLocation(bonusProg, gl.Str("uResolution\x00"))
	r.bonusUAmbient = gl.GetUniformLocation(bonusProg, gl.Str("uAmbient\x00"))
	r.bonusUSunTint = gl.GetUniformLocation(bonusProg, gl.Str("uSunTint\x00"))
	gl.Uniform1f(r.bonusUAmbient, 1.0)
	gl.Uniform3f(r.bonusUSunTint, 1.0, 1.0, 1.0)

	gl.BindVertexArray(0)
	return r, nil
}

func (r *Renderer) Destroy() {
	for _, id := range []uint32{r.chunkVBO, r.spriteVBO, r.textVBO} {
		if id != 0 {
			gl.DeleteBuffers(1, &id)
		}
	}
	for _, id := range []uint32{r.chunkVAO, r.spriteVAO, r.textVAO} {
		if id != 0 {
			gl.DeleteVertexArrays(1, &id)
		}
	}
	for _, id := range []uint32{r.chunkProg, r.spriteProg, r.glowProg, r.npcProg, r.bonusProg, r.textProg} {
		if id != 0 {
			gl.DeleteProgram(id)
		}
	}
	if r.fontTex != 0 {
		gl.DeleteTextures(1, &r.fontTex)
	}
}

func (r *Renderer) BeginFrame(cam Camera, fbW, fbH int) {
	gl.Viewport(0, 0, int32(fbW), int32(fbH))
	gl.Clear(gl.COLOR_BUFFER_BIT)

	// Set up chunk program as default for the frame.
	gl.UseProgram(r.chunkProg)
	gl.BindVertexArray(r.chunkVAO)

	gl.Uniform2f(r.uCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(r.uZoom, float32(cam.Zoom))
	gl.Uniform2f(r.uResolution, float32(fbW), float32(fbH))
	gl.Uniform2f(r.uChunkSize, float32(ChunkSize), float32(ChunkSize))
	gl.Uniform1f(r.uRotation, 0)

	gl.ActiveTexture(gl.TEXTURE0)
}

// SetSunLight sets the ambient multiplier and color tint on chunk, sprite, and bonus programs.
func (r *Renderer) SetSunLight(ambient, tintR, tintG, tintB float32) {
	gl.UseProgram(r.chunkProg)
	gl.Uniform1f(r.chunkUAmbient, ambient)
	gl.Uniform3f(r.chunkUSunTint, tintR, tintG, tintB)

	gl.UseProgram(r.spriteProg)
	gl.Uniform1f(r.spUAmbient, ambient)
	gl.Uniform3f(r.spUSunTint, tintR, tintG, tintB)

	gl.UseProgram(r.bonusProg)
	gl.Uniform1f(r.bonusUAmbient, ambient)
	gl.Uniform3f(r.bonusUSunTint, tintR, tintG, tintB)

	// Restore chunk program as active.
	gl.UseProgram(r.chunkProg)
	gl.BindVertexArray(r.chunkVAO)
}

// SetSpriteAmbient overrides ambient on the sprite program only (for glow bypass).
func (r *Renderer) SetSpriteAmbient(ambient, tintR, tintG, tintB float32) {
	gl.UseProgram(r.spriteProg)
	gl.Uniform1f(r.spUAmbient, ambient)
	gl.Uniform3f(r.spUSunTint, tintR, tintG, tintB)
}
