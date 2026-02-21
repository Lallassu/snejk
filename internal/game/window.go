//go:build !android

package game

import (
	"fmt"

	"github.com/go-gl/glfw/v3.3/glfw"
)

func initWindow() (*glfw.Window, error) {
	if err := glfw.Init(); err != nil {
		return nil, fmt.Errorf("glfw init: %w", err)
	}

	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)
	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.Decorated, glfw.True)
	glfw.WindowHint(glfw.Floating, glfw.True)

	window, err := glfw.CreateWindow(WindowWidth, WindowHeight, "Qake The Snake", nil, nil)
	if err != nil {
		return nil, fmt.Errorf("create window: %w", err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	return window, nil
}
