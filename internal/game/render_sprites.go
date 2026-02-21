//go:build !android

package game

import "github.com/go-gl/gl/v4.1-core/gl"

// DrawSprites renders an array of point sprites using the sprite program.
// buf format: [x, y, size, r, g, b, a, rotation] * N (8 floats per sprite).
// blend: true = standard alpha blend, false = additive (glow).
func (r *Renderer) DrawSprites(buf []float32, cam Camera, fbW, fbH int, additive bool) {
	if len(buf) == 0 {
		return
	}

	count := len(buf) / 8
	if count > MaxParticleRender {
		count = MaxParticleRender
	}

	gl.UseProgram(r.spriteProg)
	gl.BindVertexArray(r.spriteVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.spriteVBO)

	gl.Uniform2f(r.spUCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(r.spUZoom, float32(cam.Zoom))
	gl.Uniform2f(r.spUResolution, float32(fbW), float32(fbH))

	gl.Enable(gl.BLEND)
	if additive {
		gl.BlendFunc(gl.ONE, gl.ONE)
	} else {
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}

	gl.BufferData(gl.ARRAY_BUFFER, count*8*4, gl.Ptr(buf), gl.STREAM_DRAW)
	gl.DrawArrays(gl.POINTS, 0, int32(count))

	gl.Disable(gl.BLEND)
}

// DrawGlowSprites renders light sprites with additive blending and radial falloff.
// buf format: same as DrawSprites — [x, y, size, r, g, b, a, rotation] * N.
// RGB values should be pre-multiplied by desired brightness.
func (r *Renderer) DrawGlowSprites(buf []float32, cam Camera, fbW, fbH int) {
	if len(buf) == 0 {
		return
	}
	count := len(buf) / 8
	if count > MaxParticleRender {
		count = MaxParticleRender
	}
	gl.UseProgram(r.glowProg)
	gl.BindVertexArray(r.spriteVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.spriteVBO)
	gl.Uniform2f(r.glowUCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(r.glowUZoom, float32(cam.Zoom))
	gl.Uniform2f(r.glowUResolution, float32(fbW), float32(fbH))
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.ONE, gl.ONE)
	gl.BufferData(gl.ARRAY_BUFFER, count*8*4, gl.Ptr(buf), gl.STREAM_DRAW)
	gl.DrawArrays(gl.POINTS, 0, int32(count))
	gl.Disable(gl.BLEND)
}

// DrawBonusSprites renders bonus pickup boxes using the rotated-box shader.
// buf format: same as DrawSprites — [x, y, size, r, g, b, a, rotation] * N.
func (r *Renderer) DrawBonusSprites(buf []float32, cam Camera, fbW, fbH int) {
	if len(buf) == 0 {
		return
	}
	count := len(buf) / 8
	if count > MaxParticleRender {
		count = MaxParticleRender
	}

	gl.UseProgram(r.bonusProg)
	gl.BindVertexArray(r.spriteVAO)
	gl.BindBuffer(gl.ARRAY_BUFFER, r.spriteVBO)

	gl.Uniform2f(r.bonusUCamera, float32(cam.X), float32(cam.Y))
	gl.Uniform1f(r.bonusUZoom, float32(cam.Zoom))
	gl.Uniform2f(r.bonusUResolution, float32(fbW), float32(fbH))

	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	gl.BufferData(gl.ARRAY_BUFFER, count*8*4, gl.Ptr(buf), gl.STREAM_DRAW)
	gl.DrawArrays(gl.POINTS, 0, int32(count))

	gl.Disable(gl.BLEND)
}

// RestoreChunkProgram switches back to the chunk program after sprite drawing.
func (r *Renderer) RestoreChunkProgram() {
	gl.UseProgram(r.chunkProg)
	gl.BindVertexArray(r.chunkVAO)
}
