#version 410
out vec4 frag_colour;
uniform ivec2 iResolution;
uniform int iStep;
uniform float fTime;
uniform float fTimeDelta;
uniform float fBeginning;
uniform ivec4 iMouse;
uniform ivec4 iDate;

void main() {
	frag_colour = vec4(0.2, 0.2, 0.2, 1);
}