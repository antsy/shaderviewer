package main

import (
	"runtime"

	"flag"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/go-gl/gl/v4.1-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"
)

var (
	// Versioning
	BuildTime  string // Set by build script
	CommitHash string // Set by build script

	// CLI parameters
	ResetOnChange      bool
	WaitTime           int
	Width              int    // Can change if window resizes
	Height             int    // Can change if window resizes
	FragmentShaderFile string // Path to the shader file
	UseFullScreen      bool
	MonitorIndex       int

	// Used by uniforms
	BeginTime       time.Time
	RenderTime      time.Time
	StepNumber      int32
	MouseX          int
	MouseY          int
	MouseLeftClick  int
	MouseRightClick int

	// Everything else
	WindowTitle    string
	ProgramChannel chan string // When this channel has updates (from the file watcher), the shader will recompile
)

var NullTerminator = "\x00" // OpenGL wants its values to be 0 terminated

func main() {
	initHelpAndGlobals()

	log.Println("ShaderViewer")

	fragmentShaderSource := loadFragmentShader()

	go initFileWatcher(FragmentShaderFile)

	runtime.LockOSThread()

	window := initGlfw()
	defer glfw.Terminate()

	// Set listeners
	window.SetFramebufferSizeCallback(onWindowResize)
	window.SetCursorPosCallback(onMouseMove)
	window.SetMouseButtonCallback(onMouseClick)
	// TODO: close window if program terminates

	initOpenGL()

	// First time compile shader
	program := compileProgram(getVertexShader(), fragmentShaderSource)

	vertexObjectArray := constructVertexObjectArray()
	for !window.ShouldClose() {
		select {
		case fragmentShaderSource, _ := <-ProgramChannel:
			log.Println("New shader program received")
			program = compileProgram(getVertexShader(), fragmentShaderSource)
		default:
			draw(vertexObjectArray, window, program)
		}
	}
}

/**
 * Initialize help text and set some initial values to our globals.
 */
func initHelpAndGlobals() {
	flag.IntVar(&Width, "x", 640,
		"Width of the output window")
	flag.IntVar(&Height, "y", 400,
		"Height of the output window")
	flag.StringVar(&FragmentShaderFile, "f", "default.frag",
		"Source file for fragment shader")
	flag.BoolVar(&ResetOnChange, "r", false,
		"Reset the timer (iStep and fTime) values whenever shader is recompiled")
	flag.IntVar(&WaitTime, "w", 0,
		"Apply X milliseconds of sleeping time between each rendering")
	var generateTemplate bool
	flag.BoolVar(&generateTemplate, "generate", false,
		"Instead of running, generate empty template fragment shader file")
	flag.BoolVar(&UseFullScreen, "f11", false,
		`Use full screen mode instead of window. Uses system primary monitor.
Uses monitor's own resolution if the 'x' and 'y' -parameters are not specified.`)
	flag.IntVar(&MonitorIndex, "m", -1,
		"Instead of system primary monitor, try to use monitor by index value [0...N].")
	var listMonitors bool
	flag.BoolVar(&listMonitors, "monitors", false,
		"Instead of running, display list of monitors available")

	currentExecutableName := filepath.Base(os.Args[0])
	helpText := fmt.Sprintf(`Usage: %s [OPTION]...

Display a fragment shader in a window.
The vertex shader is provided internally from the program.
It simply fills the entire screen with a single polygon, which then can be used as a canvas.

  Executable information:
    Build time %s
    Git hash %s

  Uniforms available for the shader
    int	   iStep        Running render frame count
    vec3   iResolution  Window pixel resolution (width and height)
    float  fTime        Current running time in milliseconds
                        Note that this is affected by -r flag
    float  fTimeDelta   Time passed in milliseconds since last frame was rendered
    float  fTimestamp   Current UNIX timestamp in milliseconds
    vec4   iDate        Year, month, day as values and time of day in total seconds
    vec4   iMouse       Mouse pixel coordinates x and y, first and second mouse button states
                        1 if mouse button is pressed, 0 if lifted

OPTION(s):
`, currentExecutableName, BuildTime, CommitHash)

	flag.Usage = func() {
		fmt.Println(helpText)
		flag.PrintDefaults()
	}

	flag.Parse()

	WindowTitle = fmt.Sprintf("%s - ShaderViewer", FragmentShaderFile)

	if listMonitors {
		listSystemMonitors()
		os.Exit(0)
	}

	if generateTemplate {
		generateTemplateFile()
		os.Exit(0) // Get out!
	}

	ProgramChannel = make(chan string)

	BeginTime = time.Now()
}

/**
 * Bind the input values and draw the shader to the window
 */
func draw(vertexObjectArray uint32, window *glfw.Window, program uint32) {
	deltaTime := float32(time.Since(RenderTime)) // some loss of precision here
	RenderTime = time.Now()
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.UseProgram(program)

	resolutionUniform := gl.GetUniformLocation(program, gl.Str("iResolution"+NullTerminator))
	gl.Uniform2i(resolutionUniform, int32(Width), int32(Height))

	stepUniform := gl.GetUniformLocation(program, gl.Str("iStep"+NullTerminator))
	gl.Uniform1i(stepUniform, StepNumber)

	deltaTimeUniform := gl.GetUniformLocation(program, gl.Str("fTimeDelta"+NullTerminator))
	gl.Uniform1f(deltaTimeUniform, deltaTime)

	timestampUniform := gl.GetUniformLocation(program, gl.Str("fTimestamp"+NullTerminator))
	gl.Uniform1f(timestampUniform, float32(RenderTime.UnixNano()/int64(time.Millisecond)))

	timeUniform := gl.GetUniformLocation(program, gl.Str("fTime"+NullTerminator))
	gl.Uniform1f(timeUniform, float32((RenderTime.UnixNano()-BeginTime.UnixNano())/int64(time.Millisecond)))

	dateUniform := gl.GetUniformLocation(program, gl.Str("iDate"+NullTerminator))
	timeInSeconds := int32(RenderTime.Hour()*int(time.Hour) + RenderTime.Minute()*int(time.Minute) + RenderTime.Second())
	gl.Uniform4i(dateUniform, int32(RenderTime.Year()), int32(RenderTime.Month()), int32(RenderTime.Day()), timeInSeconds)

	mouseStateUniform := gl.GetUniformLocation(program, gl.Str("iMouse"+NullTerminator))
	gl.Uniform4i(mouseStateUniform, int32(MouseX), int32(MouseY), int32(MouseLeftClick), int32(MouseRightClick))

	gl.BindVertexArray(vertexObjectArray)
	gl.DrawArrays(gl.TRIANGLES, 0, int32(3))

	glfw.PollEvents()
	window.SwapBuffers()
	StepNumber++
	if WaitTime > 0 {
		time.Sleep(time.Duration(WaitTime) * time.Millisecond)
	}
}

/**
 * Initialize OpenGL and compile shader code.
 */
func initOpenGL() {
	if err := gl.Init(); err != nil {
		log.Println("Unable to initialize OpenGL.")
		panic(err)
	}
	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)
}

/**
 * Compiles new shader program.
 */
func compileProgram(vertexShaderSource string, fragmentShaderSource string) uint32 {
	vertexShader := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	fragmentShader := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)

	program := gl.CreateProgram()
	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	if ResetOnChange {
		log.Println("Reset timer(s)")
		BeginTime = time.Now()
		StepNumber = 0
	}

	return program
}

/**
 * Compile shader.
 */
func compileShader(source string, shaderType uint32) uint32 {
	shader := gl.CreateShader(shaderType)

	cSources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, cSources, nil)
	gl.CompileShader(shader)
	free() // When was the last time you had to manage your memory yourself?

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		compileLog := strings.Repeat(NullTerminator, int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(compileLog))

		log.Println("Unable to compile shader, error(s):")
		log.Println("\n" + compileLog)

		return 0
	}

	return shader
}

/**
 * initialize GLFW.
 */
func initGlfw() *glfw.Window {
	if err := glfw.Init(); err != nil {
		log.Println("Unable to initialize GLFW.")
		panic(err)
	}

	glfw.WindowHint(glfw.Resizable, glfw.True)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 1)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	logResolution()
	var monitor *glfw.Monitor
	if UseFullScreen {
		monitor = getMonitor()
	}
	window, err := glfw.CreateWindow(Width, Height, WindowTitle, monitor, nil)
	if err != nil {
		log.Println("Unable to initialize window for GLFW.")
		panic(err)
	}
	window.MakeContextCurrent()

	return window
}

/**
 * Briefly initialise GLFW and list monitor(s) available in the system.
 */
func listSystemMonitors() {
	if err := glfw.Init(); err != nil {
		log.Println("Unable to initialize GLFW.")
		panic(err)
	}

	monitors := glfw.GetMonitors()
	tw := tabwriter.NewWriter(os.Stdout, 6, 0, 2, ' ', tabwriter.TabIndent)
	fmt.Fprintln(tw, "Index\tName\tResolution\tRefresh rate\tWidth\tHeight")
	for index, monitor := range monitors {
		//monitor.GetWin32Monitor()
		//monitor.GetWin32Adapter()
		videoMode := monitor.GetVideoMode()
		physicalWidth, physicalHeight := monitor.GetPhysicalSize()
		fmt.Fprintln(tw, fmt.Sprintf(
			"[%d]\t%s\t%dx%d\t%dHz\t%dmm\t%dmm",
			index,
			monitor.GetName(),
			videoMode.Width,
			videoMode.Height,
			videoMode.RefreshRate,
			physicalWidth,
			physicalHeight))
	}
	tw.Flush()
}

/**
 * Determine correct monitor settings to use for full screen mode.
 */
func getMonitor() *glfw.Monitor {
	monitor := glfw.GetPrimaryMonitor()
	if MonitorIndex > -1 {
		monitors := glfw.GetMonitors()
		if !(len(monitors) < MonitorIndex+1) {
			monitor = monitors[MonitorIndex]
		} else {
			log.Println(fmt.Sprintf(
				"System has only %d monitors, monitor index %d is out of range, falling back to default monitor.",
				len(monitors),
				MonitorIndex),
			)
		}
	}

	log.Println(fmt.Sprintf(`Using monitor "%s""`, monitor.GetName()))
	videoMode := monitor.GetVideoMode()
	monitorWidth := videoMode.Width
	monitorHeight := videoMode.Height
	if !(isFlagPassed("x") && isFlagPassed("y")) {
		log.Println(fmt.Sprintf("Switching to monitor's native resolution (%dx%d)", monitorWidth, monitorHeight))
		Width = monitorWidth
		Height = monitorHeight
	}

	logResolution()

	return monitor
}

/**
 * Check if flag was passed as a parameter by user.
 */
func isFlagPassed(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

/**
 * Initialize file watcher.
 */
func initFileWatcher(filePath string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("Unable to setup file watcher: ", err)
		return
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("File watcher event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("File was modified -", event.Name)
					fragmentShaderSource := loadFragmentShader()
					ProgramChannel <- fragmentShaderSource
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("File watcher error:", err)
			}
		}
	}()

	err = watcher.Add(filePath)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

/**
 * Load shader code from a file into a string.
 */
func loadFragmentShader() string {
	log.Println(fmt.Sprintf("Loading new fragment shader file '%s'", FragmentShaderFile))
	content, err := ioutil.ReadFile(FragmentShaderFile)
	if err != nil {
		log.Println(fmt.Sprintf("Unable to read source file '%s'", FragmentShaderFile))
		log.Fatal(err)
	}

	return string(content) + NullTerminator
}

/**
 * Creates a dummy vertex object array containing single point.
 */
func constructVertexObjectArray() uint32 {
	points := []float32{0, 0, 0}
	var vertexBufferObject uint32
	gl.GenBuffers(1, &vertexBufferObject)
	gl.BindBuffer(gl.ARRAY_BUFFER, vertexBufferObject)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vertexObjectArray uint32
	gl.GenVertexArrays(1, &vertexObjectArray)
	gl.BindVertexArray(vertexObjectArray)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vertexObjectArray)
	gl.VertexAttribPointer(0, 1, gl.FLOAT, false, 0, nil)

	return vertexObjectArray
}

/**
 * Sets viewport size to match window frame buffer size.
 */
func onWindowResize(window *glfw.Window, w int, h int) {
	w, h = window.GetFramebufferSize()
	Width = w
	Height = h
	gl.Viewport(0, 0, int32(Width), int32(Height))
	logResolution()
}

/**
 * Record mouse position over the window.
 */
func onMouseMove(window *glfw.Window, x float64, y float64) {
	MouseX = int(x)
	MouseY = Height - int(y) // Because old school dudes always play with their mouse inverted
}

/**
 * Record mouse clicks when mouse event occurs.
 *
 * @param	window		Window handle
 * @param	button		Which mouse button was clicked
 * @param	action		1 = mouse button down, 0 = mouse button up
 * @param	mods		Modifier key is pressed? 1=shift, 2=ctrl etc.
 */
func onMouseClick(window *glfw.Window, button glfw.MouseButton, action glfw.Action, mods glfw.ModifierKey) {
	MouseLeftClick = boolToInt(button == glfw.MouseButton1 && action == glfw.Press)
	MouseRightClick = boolToInt(button == glfw.MouseButton2 && action == glfw.Press)
}

/**
 * Convert boolean value to integer.
 */
func boolToInt(boolean bool) int {
	if boolean {
		return 1
	}
	return 0
}

/**
 * Log resolution changes.
 */
func logResolution() {
	log.Println(fmt.Sprintf("Setting resolution to %dx%d", Width, Height))
}

/**
 * Vertex shader that only fills the screen with a single polygon.
 */
func getVertexShader() string {
	return `#version 410
out vec2 texCoord;

void main() {
	float x = -1.0 + float((gl_VertexID & 1) << 2);
	float y = -1.0 + float((gl_VertexID & 2) << 1);
	texCoord.x = (x + 1.0) * 0.5;
	texCoord.y = (y + 1.0) * 0.5;
	gl_Position = vec4(x, y, 0, 1);
}
` + NullTerminator
}

/**
 * Write out an empty template file to start coding with.
 */
func generateTemplateFile() {
	filename := "default.frag"
	template := `#version 410
out vec4 frag_colour;
uniform ivec2 iResolution;
uniform int iStep;
uniform float fTime;
uniform float fTimeDelta;
uniform float fTimestamp;
uniform ivec4 iMouse;
uniform ivec4 iDate;

void main() {
	frag_colour = vec4(0.0, 0.87, 0.0, 1);
}` // The default template has the Atari GEM green color. I find it very pretty.

	file, err := os.OpenFile(filename, syscall.O_WRONLY|syscall.O_CREAT|syscall.O_EXCL, 0666)
	if err != nil {
		log.Println("Unable to create template file")
		log.Println(err)
	} else {
		file.WriteString(template)
		log.Println(fmt.Sprintf("File '%s' has been written for you", filename))
	}
	file.Close()
}
