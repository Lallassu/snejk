//go:build !android

package game

import (
	"math"

	"github.com/go-gl/glfw/v3.3/glfw"
)

type Input struct {
	prevMouse   map[glfw.MouseButton]bool
	prevKeys    map[glfw.Key]bool
	prevCursorX float64
	prevCursorY float64
}

func NewInput() *Input {
	return &Input{
		prevMouse: make(map[glfw.MouseButton]bool),
		prevKeys:  make(map[glfw.Key]bool),
	}
}

func (in *Input) JustPressed(window *glfw.Window, key glfw.Key) bool {
	down := window.GetKey(key) == glfw.Press
	jp := down && !in.prevKeys[key]
	in.prevKeys[key] = down
	return jp
}

func (in *Input) JustClicked(window *glfw.Window, btn glfw.MouseButton) bool {
	down := window.GetMouseButton(btn) == glfw.Press
	jp := down && !in.prevMouse[btn]
	in.prevMouse[btn] = down
	return jp
}

// CursorWorldPos converts cursor position to world coordinates.
func CursorWorldPos(window *glfw.Window, cam Camera, fbW, fbH int) (float64, float64) {
	cx, cy := window.GetCursorPos()
	winW, winH := window.GetSize()
	if winW <= 0 || winH <= 0 {
		return cam.X, cam.Y
	}
	scaleX := float64(fbW) / float64(winW)
	scaleY := float64(fbH) / float64(winH)
	fx := cx * scaleX
	fy := cy * scaleY
	wx := cam.X + (fx-float64(fbW)*0.5)/cam.Zoom
	wy := cam.Y + (fy-float64(fbH)*0.5)/cam.Zoom
	return wx, wy
}

// SnakeSteerTarget returns the desired heading angle and whether the snake is idle.
// Idle = no WASD input, cursor hasn't moved this frame, and cursor is within the deadzone.
// WASD gives cardinal directions and always overrides idle.
func SnakeSteerTarget(window *glfw.Window, in *Input, snake *Snake, cam Camera, fbW, fbH int) (float64, bool) {
	if snake == nil {
		return 0, false
	}

	// WASD: cardinal directions take priority — never idle while steering by key.
	if window.GetKey(glfw.KeyW) == glfw.Press {
		return -math.Pi / 2, false
	}
	if window.GetKey(glfw.KeyS) == glfw.Press {
		return math.Pi / 2, false
	}
	if window.GetKey(glfw.KeyA) == glfw.Press {
		return math.Pi, false
	}
	if window.GetKey(glfw.KeyD) == glfw.Press {
		return 0, false
	}

	// Track cursor movement (window pixel space).
	cx, cy := window.GetCursorPos()
	cursorMoved := math.Hypot(cx-in.prevCursorX, cy-in.prevCursorY) > 0.5
	in.prevCursorX, in.prevCursorY = cx, cy

	// Mouse: steer toward cursor when it's far enough from the head.
	hx, hy := snake.Head()
	mx, my := CursorWorldPos(window, cam, fbW, fbH)
	dx := mx - hx
	dy := my - hy
	dist := math.Hypot(dx, dy)

	// Reduced deadzone (1.5) so snake can reach bonuses under the cursor.
	// The bonus collection radius is 3.0, so a smaller deadzone ensures the
	// snake keeps moving toward the target until very close.
	if dist >= 1.5 {
		return math.Atan2(dy, dx), false
	}

	// Cursor is within the deadzone (essentially on the head). Idle if also not moving.
	if !cursorMoved {
		return snake.Heading, true
	}
	return snake.Heading, false
}

// UpdateCameraZoom handles E/R zoom only (no panning — camera follows snake).
func UpdateCameraZoom(cam *Camera, window *glfw.Window, dt float64, fbW, fbH int) {
	zoomRate := 1.4
	if window.GetKey(glfw.KeyE) == glfw.Press {
		cam.Zoom *= math.Exp(zoomRate * dt)
	}
	if window.GetKey(glfw.KeyR) == glfw.Press {
		cam.Zoom *= math.Exp(-zoomRate * dt)
	}
	cam.Clamp(fbW, fbH)
}
