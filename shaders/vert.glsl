#version 430
in layout(location = 0) vec4 position;
layout(location = 1) uniform mat4 projMat;
layout(location = 2) uniform mat4 viewMat;
out vec4 col;

float frac(float x) {
    return x - floor(x);
}

void main() {
  gl_Position = projMat * viewMat * position;
  float id = gl_VertexID;
  col = vec4(frac(id/200000), frac(id/300000), frac(id/400000), 1);
}
