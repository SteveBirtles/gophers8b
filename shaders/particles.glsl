#version 430 core
#define NUMPARTICLES 1000000
#define ATTRACTORS 3

uniform ParticleDataBlock {
    float t;
    float[ATTRACTORS] x;
    float[ATTRACTORS] y;
    float[ATTRACTORS] g;
};

layout (local_size_x = 1024, local_size_y = 1) in;

layout (std140, binding = 0) buffer Pos {
  vec4 positions[];
};

layout (std140, binding = 1) buffer Vel {
  vec4 velocities[];
};

void main() {
  uint index = gl_GlobalInvocationID.x + gl_GlobalInvocationID.y * gl_NumWorkGroups.x * gl_WorkGroupSize.x;

	if(index > NUMPARTICLES) {
    return;
  }

  for (int i = 0; i < ATTRACTORS; i++) {

      vec3 pPos = positions[index].xyz;
      vec3 vPos = velocities[index].xyz;
      vec3 centre = vec3(x[i], y[i], 0);

      float d = distance(pPos, centre);

      if (d > 1) {
          vec3 g = (pPos-centre) * g[i];
          g = g * 1 / (pow(d, 2));
          vec3 pp = pPos + vPos * t + 0.5 * t * t * g;
          vec3 vp = vPos + g * t;
          positions[index] = vec4(pp, 1.0);
          velocities[index] = vec4(vp, 0.0);
      } else {
          velocities[index] = vec4(vPos*(1-t), 0.0);
      }

  }

}