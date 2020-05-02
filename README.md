# Shader Viewer

This software compiles OpenGL fragment shader and displays it in a window. Program will recompile the shader automatically whenever shared source file changes. ⚙

There are probably dozens of better tools for doing the same things as this one does and more. I built this for myself as a starting point for experimenting with ray marching visualizations. 🔮

If you use this software to create something cool, please let me know. 😍

If you find bugs, please fix them and send in a pull request. I'm pretty sure there is at least some memory leak somewhere. 🛠️


## Build instructions

### Windows

[GLFW](https://github.com/go-gl/glfw) might require GCC compiler, you can install one from [tdm-gcc](https://jmeubank.github.io/tdm-gcc/).

Run `make deps` to install required modules.

Then run `make compile` to compile binary.
 

### Linux

TODO. 👨‍💻


## Further reading

 * [OpenGL specs](https://www.khronos.org/registry/OpenGL/index_gl.php)
 * [OpenGL wiki](https://www.khronos.org/opengl/wiki/Main_Page)
 * [GLFW API](https://www.glfw.org/docs/3.3/) library used in this software to do everything
 * [hg_sdf](http://mercury.sexy/hg_sdf/) very helpful library for building signed distance functions
 
 