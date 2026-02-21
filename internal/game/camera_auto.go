package game

import "math"

// UpdateAutoCamera fits the entire world on screen at all times.
// Zoom is computed so the world fills the framebuffer; camera is centered on the world.
func UpdateAutoCamera(cam *Camera, snake *Snake, dt float64, fbW, fbH int) {
	zoomW := float64(fbW) / float64(WorldWidth)
	zoomH := float64(fbH) / float64(WorldHeight)
	cam.Zoom = math.Min(zoomW, zoomH)
	cam.X = float64(WorldWidth) / 2
	cam.Y = float64(WorldHeight) / 2
}
