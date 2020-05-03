#version 410
out vec4 frag_colour;
uniform ivec2 iResolution;
uniform int iStep;
uniform float fTime;
uniform float fTimeDelta;
uniform float fTimestamp;
uniform ivec4 iMouse;
uniform ivec4 iDate;

/*
	" Most people are other people.
	  Their thoughts are someone else's opinions,
	  their lives a mimicry,
	  their passions a quotation. "

	This shader was built imitating Martijn Steinrucken (aka. @The_ArtOfCode) tutorial.
*/

#define MAXIMUM_STEPS 100
#define SURFACE_DISTANCE 0.01
#define MAXIMUM_DISTANCE 100.0

float GetDistance(vec3 pointOrigin) {
	vec3 spherePosition = vec3(0, 1, 6);
	float sphereRadius = 1;
	vec4 sphere = vec4(spherePosition, sphereRadius);

	float distanceToSphere = length(pointOrigin - sphere.xyz) - sphere.w;
	float distanceToPlane = pointOrigin.y;

	float distance = min(distanceToSphere, distanceToPlane);

	return distance;
}

float RayMarch(vec3 rayOrigin, vec3 rayDirection) {
	float distanceFromOrigin = 0.0;
	for(int marchStep = 0; marchStep < MAXIMUM_STEPS; marchStep++) {
		vec3 marchingPoint = rayOrigin + distanceFromOrigin * rayDirection;
		float distanceToScene = GetDistance(marchingPoint);
		distanceFromOrigin += distanceToScene;
		if (distanceToScene < SURFACE_DISTANCE || distanceFromOrigin > MAXIMUM_DISTANCE) {
			break;
		}
	}
	return distanceFromOrigin;
}

vec3 GetNormal(vec3 point) {
	float pointDistance = GetDistance(point);
	vec2 e = vec2(.01, 0);

	vec3 n = pointDistance - vec3(
		GetDistance(point - e.xyy),
		GetDistance(point - e.yxy),
		GetDistance(point - e.yyx)
	);

	return normalize(n);
}

float GetLight(vec3 point) {
	float lightHeight = 4 + 2 * sin(fTime * 0.002); // Move light up and down periodically
	vec3 lightPosition = vec3(0, 5, lightHeight);
	lightPosition.xz += vec2(sin(fTime * 0.01), cos(fTime * 0.01)); // Circle light around the ball

	vec3 light = normalize(lightPosition - point);
	vec3 pointNormal = GetNormal(point);

	float diffuseLight = clamp(dot(pointNormal, light), 0., 1.);

	// Also render shadow
	float lightFix = SURFACE_DISTANCE * 2.0;
	float shadow = RayMarch(point + pointNormal * lightFix, light);
	if (shadow < length(lightPosition - point)) {
		diffuseLight *= 0.1;
	}

	return diffuseLight;
}

void main() {
	// Normalized coordinates
	vec2 uv = (gl_FragCoord.xy - 0.5 * iResolution.xy) / iResolution.y;

	// Setup camera
	vec3 rayOrigin = vec3(0,1,0);
	vec3 rayDirection = normalize(vec3(uv.x, uv.y, 1));

	float rayDistance = RayMarch(rayOrigin, rayDirection);

	vec3 lightingPoint = rayOrigin + rayDirection * rayDistance;
	float diffuseLight = GetLight(lightingPoint);
	vec3 color = vec3(diffuseLight);

	// Render point
	frag_colour = vec4(color, 1);
}
