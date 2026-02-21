//go:build !android

package game

import (
	"fmt"
	"strings"

	"github.com/go-gl/gl/v4.1-core/gl"
)

// Chunk vertex shader: VBO-based quad (no gl_VertexID).
const chunkVertSrc = `#version 410 core

layout(location = 0) in vec2 aPos; // 0..1 quad vertex

uniform vec2 uChunkOrigin;
uniform vec2 uChunkSize;
uniform float uRotation;
uniform vec2 uCamera;
uniform float uZoom;
uniform vec2 uResolution;

out vec2 vUV;

void main() {
    vUV = aPos;
    vec2 centre = uChunkOrigin + uChunkSize * 0.5;
    vec2 local = (aPos - 0.5) * uChunkSize;
    float c = cos(uRotation);
    float s = sin(uRotation);
    vec2 rot = vec2(c * local.x - s * local.y, s * local.x + c * local.y);
    vec2 worldPos = centre + rot;
    vec2 screenPos = (worldPos - uCamera) * uZoom + uResolution * 0.5;
    vec2 ndc = (screenPos / uResolution) * 2.0 - 1.0;
    ndc.y = -ndc.y;
    gl_Position = vec4(ndc, 0.0, 1.0);
}
` + "\x00"

// Chunk fragment shader: sample texture, multiply RGB by alpha shade factor + sun cycle.
const chunkFragSrc = `#version 410 core

uniform sampler2D uTex;
uniform float uAmbient;
uniform vec3 uSunTint;

in vec2 vUV;
out vec4 FragColor;

void main() {
    vec4 t = texture(uTex, vUV);
    float shade = t.a;
    FragColor = vec4(t.rgb * shade * uAmbient * uSunTint, 1.0);
}
` + "\x00"

// Particle vertex shader: point sprites with per-vertex pos/size/color/rotation.
const particleVertSrc = `#version 410 core

layout(location = 0) in vec2 aWorldPos;
layout(location = 1) in float aSize;
layout(location = 2) in vec4 aColor;
layout(location = 3) in float aRotation;

uniform vec2 uCamera;
uniform float uZoom;
uniform vec2 uResolution;

out vec4 vColor;
out float vRotation;

void main() {
    vec2 screenPos = (aWorldPos - uCamera) * uZoom + uResolution * 0.5;
    vec2 ndc = (screenPos / uResolution) * 2.0 - 1.0;
    ndc.y = -ndc.y;
    gl_Position = vec4(ndc, 0.0, 1.0);
    float ps = floor(aSize * uZoom + 0.5);
    gl_PointSize = max(1.0, ps);
    vColor = aColor;
    vRotation = aRotation;
}
` + "\x00"

// Particle fragment shader: solid square point sprite with sun cycle.
const particleFragSrc = `#version 410 core

uniform float uAmbient;
uniform vec3 uSunTint;

in vec4 vColor;
out vec4 FragColor;

void main() {
    FragColor = vec4(vColor.rgb * uAmbient * uSunTint, vColor.a);
}
` + "\x00"

// Glow fragment shader: additive radial falloff for light sprites.
// vColor.rgb should be pre-multiplied by desired brightness. No ambient/tint applied.
const glowFragSrc = `#version 410 core

in vec4 vColor;
out vec4 FragColor;

void main() {
    float dist = length(gl_PointCoord - vec2(0.5)) * 2.0; // 0=center, 1=edge
    float falloff = clamp(1.0 - dist, 0.0, 1.0);
    falloff = falloff * falloff; // quadratic: natural light falloff
    FragColor = vec4(vColor.rgb * falloff, 1.0);
}
` + "\x00"

// NPC fragment shader: rotated textured point sprite (for cars).
const npcFragSrc = `#version 410 core

uniform sampler2D uCarTex;
uniform float uCarAspect;

in vec4 vColor;
in float vRotation;
out vec4 FragColor;

void main() {
    vec2 uv = gl_PointCoord - vec2(0.5);
    float c = cos(vRotation);
    float s = sin(vRotation);
    vec2 rot = vec2(c * uv.x - s * uv.y, s * uv.x + c * uv.y);
    rot.x *= uCarAspect;
    uv = rot + vec2(0.5);
    vec4 t = texture(uCarTex, uv);
    vec3 col = t.rgb * vColor.rgb;
    float a = t.a * vColor.a;
    if (a < 0.01) discard;
    FragColor = vec4(col, a);
}
` + "\x00"

// Bonus box fragment shader: renders a rotated filled square with dark border and 3D bevel.
// Uses vRotation to spin the box; uv-based bevel gives a crate/pickup-box look.
const bonusFragSrc = `#version 410 core

uniform float uAmbient;
uniform vec3 uSunTint;

in vec4 vColor;
in float vRotation;
out vec4 FragColor;

void main() {
    vec2 uv = gl_PointCoord - vec2(0.5);

    float c = cos(vRotation);
    float s = sin(vRotation);
    vec2 rot = vec2(c * uv.x - s * uv.y, s * uv.x + c * uv.y);

    float outer = 0.44; // box outer edge
    float inner = 0.34; // box fill edge (border width = outer - inner)

    float ax = abs(rot.x);
    float ay = abs(rot.y);

    if (ax > outer || ay > outer) discard;

    vec3 col;
    float alpha = vColor.a;

    if (ax > inner || ay > inner) {
        // Dark border gives the box a defined edge.
        col = vec3(0.04, 0.04, 0.04);
    } else {
        col = vColor.rgb;
        // Top-left bevel highlight.
        float hiX = max(0.0, -rot.x - 0.04);
        float hiY = max(0.0, -rot.y - 0.04);
        float hi = clamp((hiX + hiY) * 2.2, 0.0, 0.5);
        col = mix(col, vec3(1.0), hi);
        // Bottom-right bevel shadow.
        float shX = max(0.0, rot.x - 0.04);
        float shY = max(0.0, rot.y - 0.04);
        float sh = clamp((shX + shY) * 1.8, 0.0, 0.35);
        col = mix(col, vec3(0.0), sh);
    }

    FragColor = vec4(col * uAmbient * uSunTint, alpha);
}
` + "\x00"

// Text vertex shader: screen-space textured quads for font rendering.
const textVertSrc = `#version 410 core

layout(location = 0) in vec2 aPos;
layout(location = 1) in vec2 aUV;
layout(location = 2) in vec4 aColor;

uniform vec2 uResolution;

out vec2 vUV;
out vec4 vColor;

void main() {
    vec2 ndc = (aPos / uResolution) * 2.0 - 1.0;
    ndc.y = -ndc.y;
    gl_Position = vec4(ndc, 0.0, 1.0);
    vUV = aUV;
    vColor = aColor;
}
` + "\x00"

// Text fragment shader: font atlas sampling with color tint.
const textFragSrc = `#version 410 core

uniform sampler2D uFontTex;

in vec2 vUV;
in vec4 vColor;
out vec4 FragColor;

void main() {
    vec4 t = texture(uFontTex, vUV);
    if (t.a < 0.01) discard;
    FragColor = vec4(t.rgb * vColor.rgb, t.a * vColor.a);
}
` + "\x00"

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)
	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLen)
		buf := strings.Repeat("\x00", int(logLen+1))
		gl.GetShaderInfoLog(shader, logLen, nil, gl.Str(buf))
		gl.DeleteShader(shader)
		return 0, fmt.Errorf("compile shader: %s", strings.TrimRight(buf, "\x00"))
	}
	return shader, nil
}

func linkProgram(vertSrc, fragSrc string) (uint32, error) {
	vs, err := compileShader(vertSrc, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}
	fs, err := compileShader(fragSrc, gl.FRAGMENT_SHADER)
	if err != nil {
		gl.DeleteShader(vs)
		return 0, err
	}

	program := gl.CreateProgram()
	gl.AttachShader(program, vs)
	gl.AttachShader(program, fs)
	gl.LinkProgram(program)

	gl.DetachShader(program, vs)
	gl.DetachShader(program, fs)
	gl.DeleteShader(vs)
	gl.DeleteShader(fs)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLen int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLen)
		buf := strings.Repeat("\x00", int(logLen+1))
		gl.GetProgramInfoLog(program, logLen, nil, gl.Str(buf))
		gl.DeleteProgram(program)
		return 0, fmt.Errorf("link program: %s", strings.TrimRight(buf, "\x00"))
	}
	return program, nil
}
