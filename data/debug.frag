#version 410
out vec4 frag_colour;
uniform ivec2 iResolution;
uniform int iStep;
uniform float fTime;
uniform float fTimeDelta;
uniform ivec4 iMouse;

void main() {
	float r = 1.0 / float(iResolution.y) * gl_FragCoord.y;
	float b = 1.0 / float(iResolution.x) * gl_FragCoord.x;
	// Color shift so we detect the steps
	float g = 1.0 / (iStep * 0.001);

	// Calculate distance to the mouse cursor
	float md = distance(iMouse.xy, gl_FragCoord.xy);

	if (md < 20 && md > 15) {
		// Draw a ring around the mouse position
		frag_colour = vec4(1.0-r, 1.0-g, 1.0-b, 1.0);
	} else {
		// Draw color normally
		frag_colour = vec4(r, g, b, 1.0);
	}

	if (md < 14) {
		// Button is clicked, fill the mouse circle
		if (iMouse.z == 1 && iMouse.x > gl_FragCoord.x) {
			frag_colour = vec4(1, 1, 1, 1); // Left side
		}	
		if (iMouse.w == 1 && iMouse.x < gl_FragCoord.x) {
			frag_colour = vec4(1, 1, 1, 1); // Right side
		}	
	}
}