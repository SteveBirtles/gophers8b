package main

import (
	"fmt"
	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/go-gl/glfw/v3.2/glfw"
	"github.com/go-gl/mathgl/mgl32"
	_ "image/png"
	"io/ioutil"
	"log"
	"math/rand"
	"runtime"
	"strings"
	"time"
)

const (
	windowWidth  = 1280
	windowHeight = 720
	numParticles = 1000000
	attractors   = 3
)

var (
	frames            = 0
	second            = time.Tick(time.Second)
	windowTitlePrefix = "Particles"
	vao               uint32
	frameLength       float32
)

func init() {
	runtime.LockOSThread()
	rand.Seed(time.Now().UTC().UnixNano())
}

func LoadShader(path string, shaderType uint32) uint32 {
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	source := string(bytes) + "\x00"

	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		panic(fmt.Errorf("failed to compile %v: %v", source, log))
	}

	return shader
}

func main() {

	var err error
	if err = glfw.Init(); err != nil {
		log.Fatalln("failed to initialize glfw:", err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(windowWidth, windowHeight, windowTitlePrefix, nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()

	if err = gl.Init(); err != nil {
		panic(err)
	}
	gl.ClearColor(0.0, 0.0, 0.0, 1.0)
	//glfw.SwapInterval(0)

	particleShader := LoadShader("shaders/particles.glsl", gl.COMPUTE_SHADER)
	vertexShader := LoadShader("shaders/vert.glsl", gl.VERTEX_SHADER)
	fragmentShader := LoadShader("shaders/frag.glsl", gl.FRAGMENT_SHADER)

	particleProg := gl.CreateProgram()
	gl.AttachShader(particleProg, particleShader)
	gl.LinkProgram(particleProg)
	gl.UseProgram(particleProg)

	var particleDataBuffer uint32
	particleDataIndex := gl.GetUniformBlockIndex(particleProg, gl.Str("ParticleDataBlock"+"\x00"))
	gl.UniformBlockBinding(particleProg, particleDataIndex, 1)
	gl.GenBuffers(1, &particleDataBuffer)

	posSSBO := uint32(1)
	velSSBO := uint32(2)

	var points, velocities []mgl32.Vec4

	for i := 0; i < numParticles; i++ {
		x := ((float32(i)/numParticles)*2 - 1) * float32(100)
		y := float32(110)*float32(i % 1000)/1000 - 55
		z := float32(0)
		points = append(points, mgl32.Vec4{x, y, z, 1})
	}

	gl.GenBuffers(1, &posSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, posSSBO)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, numParticles*16, gl.Ptr(points), gl.DYNAMIC_DRAW)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 0, posSSBO)

	for i := 0; i < numParticles; i++ {
		velocities = append(velocities, mgl32.Vec4{float32(0), float32(0), float32(0), 0})
	}

	gl.GenBuffers(1, &velSSBO)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, velSSBO)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, numParticles*16, gl.Ptr(velocities), gl.DYNAMIC_DRAW)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 1, velSSBO)

	attractorVectors := make([]float32, 0)

	for i := 0; i < attractors; i++ {
		attractorVectors = append(attractorVectors, (rand.Float32()*2-1)*float32(80))
	}
	for i := 0; i < attractors; i++ {
		attractorVectors = append(attractorVectors, (rand.Float32()*2-1)*float32(50))
	}
	for i := 0; i < attractors; i++ {
		attractorVectors = append(attractorVectors, -rand.Float32()*float32(20))
	}

	quadProg := gl.CreateProgram()
	gl.AttachShader(quadProg, vertexShader)
	gl.AttachShader(quadProg, fragmentShader)
	gl.LinkProgram(quadProg)

	gl.UseProgram(quadProg)

	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.BindBuffer(gl.ARRAY_BUFFER, posSSBO)
	gl.VertexAttribPointer(0, 4, gl.FLOAT, false, 0, nil)
	gl.EnableVertexAttribArray(0)

	position := mgl32.Vec3{0, 0, 100}
	target := mgl32.Vec3{0, 0, 0}
	up := mgl32.Vec3{0, 1, 0}
	view := mgl32.LookAtV(position, target, up)
	projection := mgl32.Perspective(mgl32.DegToRad(60), float32(windowWidth)/float32(windowHeight), 0.1, 1000.0)

	projUniform := int32(1)
	gl.UniformMatrix4fv(projUniform, 1, false, &projection[0])

	viewUniform := int32(2)
	gl.UniformMatrix4fv(viewUniform, 1, false, &view[0])

	for !window.ShouldClose() {

		frameStart := time.Now()

		if window.GetKey(glfw.KeyEscape) == glfw.Press {
			window.SetShouldClose(true)
		}

		/* --------------------------- */

		gl.UseProgram(particleProg)

		gl.BindBuffer(gl.UNIFORM_BUFFER, particleDataBuffer)
		particleDataBlock := []float32{frameLength}
		particleDataBlock = append(particleDataBlock, attractorVectors...)

		gl.BufferData(gl.UNIFORM_BUFFER, len(particleDataBlock)*4, gl.Ptr(particleDataBlock), gl.DYNAMIC_COPY)
		gl.BindBufferBase(gl.UNIFORM_BUFFER, 1, particleDataBuffer)

		gl.DispatchCompute(1024, 1, 1)
		gl.MemoryBarrier(gl.VERTEX_ATTRIB_ARRAY_BARRIER_BIT)

		//gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER,0, numParticles*16, gl.Ptr(points))
		//fmt.Println(points[0])

		gl.UseProgram(quadProg)
		gl.ClearColor(0, 0, 0, 1)
		gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
		gl.BindVertexArray(vao)
		gl.PointSize(3)
		gl.DrawArrays(gl.POINTS, 0, numParticles)

		/* --------------------------- */

		gl.UseProgram(0)
		window.SwapBuffers()

		glfw.PollEvents()
		frames++
		select {
		case <-second:
			window.SetTitle(fmt.Sprintf("%s | FPS: %d", windowTitlePrefix, frames))
			frames = 0
		default:
		}
		frameLength = float32(time.Since(frameStart).Seconds())

	}

	glfw.Terminate()
}
