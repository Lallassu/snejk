//go:build !android

package game

import "github.com/go-gl/gl/v4.1-core/gl"

// EnsureTexture creates a GL texture for a chunk if it doesn't have one yet.
func (r *Renderer) EnsureTexture(c *Chunk) {
	if c.Tex != 0 {
		return
	}
	gl.GenTextures(1, &c.Tex)
	gl.BindTexture(gl.TEXTURE_2D, c.Tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)

	gl.TexImage2D(
		gl.TEXTURE_2D, 0, gl.RGBA8,
		ChunkSize, ChunkSize, 0,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(c.Pixels),
	)
	c.NeedsUpload = false
}

// UploadChunk re-uploads pixel data for a chunk whose texture already exists.
func (r *Renderer) UploadChunk(c *Chunk) {
	r.EnsureTexture(c)
	gl.BindTexture(gl.TEXTURE_2D, c.Tex)
	gl.TexSubImage2D(
		gl.TEXTURE_2D, 0, 0, 0,
		ChunkSize, ChunkSize,
		gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(c.Pixels),
	)
	c.NeedsUpload = false
}

// DrawChunk renders a single chunk (assumes chunk program is active).
func (r *Renderer) DrawChunk(c *Chunk) {
	if c == nil || c.Tex == 0 {
		return
	}
	baseX, baseY := c.WorldOrigin()
	gl.Uniform2f(r.uChunkOrigin, float32(baseX), float32(baseY))
	gl.Uniform1f(r.uRotation, 0)
	gl.BindTexture(gl.TEXTURE_2D, c.Tex)
	gl.DrawArrays(gl.TRIANGLES, 0, 6)
}

// DrawChunks renders all visible chunks: recompute shadows, upload dirty, draw.
func (r *Renderer) DrawChunks(w *World, cam Camera, fbW, fbH int) {
	halfW := float64(fbW) / (2.0 * cam.Zoom)
	halfH := float64(fbH) / (2.0 * cam.Zoom)
	view := RectF{
		X0: cam.X - halfW, Y0: cam.Y - halfH,
		X1: cam.X + halfW, Y1: cam.Y + halfH,
	}

	var keys []ChunkKey
	keys = w.VisibleChunks(view, keys)

	gl.UseProgram(r.chunkProg)
	gl.BindVertexArray(r.chunkVAO)
	gl.ActiveTexture(gl.TEXTURE0)

	for _, k := range keys {
		c := w.GetChunk(k.X, k.Y)
		if c == nil {
			continue
		}
		if c.NeedsShadow {
			c.RecomputeShadows(w)
		}
		if c.NeedsUpload {
			r.UploadChunk(c)
		} else {
			r.EnsureTexture(c)
		}
		r.DrawChunk(c)
	}
}
