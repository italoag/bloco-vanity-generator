//go:build darwin && arm64 && cgo

package engine

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework Metal
#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static char* metal_copy_string(NSString *value) {
	if (value == nil) {
		return NULL;
	}
	const char *utf8 = [value UTF8String];
	if (utf8 == NULL) {
		return NULL;
	}
	return strdup(utf8);
}

static char* metal_copy_error(NSError *error) {
	if (error == nil) {
		return NULL;
	}
	return metal_copy_string([error localizedDescription]);
}

static int metal_available(void) {
	@autoreleasepool {
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		return device == nil ? 0 : 1;
	}
}

@interface BlocoMetalMatchContext : NSObject
@property(nonatomic, strong) id<MTLDevice> device;
@property(nonatomic, strong) id<MTLComputePipelineState> pipeline;
@property(nonatomic, strong) id<MTLCommandQueue> queue;
@end

@implementation BlocoMetalMatchContext
@end

static NSString* metal_match_source(void) {
	return @"#include <metal_stdlib>\n"
		"using namespace metal;\n"
		"constant ulong KECCAKF_RNDC[24] = {\n"
		"  0x0000000000000001UL, 0x0000000000008082UL, 0x800000000000808aUL, 0x8000000080008000UL,\n"
		"  0x000000000000808bUL, 0x0000000080000001UL, 0x8000000080008081UL, 0x8000000000008009UL,\n"
		"  0x000000000000008aUL, 0x0000000000000088UL, 0x0000000080008009UL, 0x000000008000000aUL,\n"
		"  0x000000008000808bUL, 0x800000000000008bUL, 0x8000000000008089UL, 0x8000000000008003UL,\n"
		"  0x8000000000008002UL, 0x8000000000000080UL, 0x000000000000800aUL, 0x800000008000000aUL,\n"
		"  0x8000000080008081UL, 0x8000000000008080UL, 0x0000000080000001UL, 0x8000000080008008UL\n"
		"};\n"
		"constant uchar KECCAKF_ROTC[25] = {\n"
		"  0, 1, 62, 28, 27,\n"
		"  36, 44, 6, 55, 20,\n"
		"  3, 10, 43, 25, 39,\n"
		"  41, 45, 15, 21, 8,\n"
		"  18, 2, 61, 56, 14\n"
		"};\n"
		"static inline ulong rotl64(ulong value, uchar shift) {\n"
		"  return shift == 0 ? value : ((value << shift) | (value >> (64 - shift)));\n"
		"}\n"
		"constant uint FE_P[16] = { 0xfc2f, 0xffff, 0xfffe, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff };\n"
		"constant uint FE_PM2[16] = { 0xfc2d, 0xffff, 0xfffe, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff, 0xffff };\n"
		"constant uint SECP_GX[16] = { 0x1798, 0x16f8, 0x815b, 0x59f2, 0x28d9, 0x2dce, 0xfcdb, 0x029b, 0x0b07, 0xce87, 0x6295, 0x55a0, 0xbbac, 0xf9dc, 0x667e, 0x79be };\n"
		"constant uint SECP_GY[16] = { 0xd4b8, 0xfb10, 0xd08f, 0x9c47, 0x5419, 0xa685, 0xb448, 0xfd17, 0x08a8, 0x0e11, 0xfbfc, 0x5da4, 0xc465, 0x26a3, 0xda77, 0x483a };\n"
		"static inline ulong load64_le_thread(thread const uchar *bytes) {\n"
		"  ulong value = 0;\n"
		"  for (uint i = 0; i < 8; i++) { value |= ((ulong)bytes[i]) << (8 * i); }\n"
		"  return value;\n"
		"}\n"
		"static inline void fe_zero(thread uint *r) { for (uint i = 0; i < 16; i++) { r[i] = 0; } }\n"
		"static inline void fe_one(thread uint *r) { fe_zero(r); r[0] = 1; }\n"
		"static inline void fe_copy(thread uint *r, thread const uint *a) { for (uint i = 0; i < 16; i++) { r[i] = a[i]; } }\n"
		"static inline bool fe_is_zero(thread const uint *a) { for (uint i = 0; i < 16; i++) { if (a[i] != 0) { return false; } } return true; }\n"
		"static inline bool fe_is_one(thread const uint *a) { if (a[0] != 1) { return false; } for (uint i = 1; i < 16; i++) { if (a[i] != 0) { return false; } } return true; }\n"
		"static inline int fe_cmp_p(thread const uint *a) {\n"
		"  for (int i = 15; i >= 0; i--) { if (a[i] > FE_P[i]) { return 1; } if (a[i] < FE_P[i]) { return -1; } }\n"
		"  return 0;\n"
		"}\n"
		"static inline void fe_sub_p(thread uint *a) {\n"
		"  uint borrow = 0;\n"
		"  for (uint i = 0; i < 16; i++) {\n"
		"    uint bi = FE_P[i] + borrow;\n"
		"    if (a[i] >= bi) { a[i] = a[i] - bi; borrow = 0; } else { a[i] = (uint)(0x10000u + a[i] - bi); borrow = 1; }\n"
		"  }\n"
		"}\n"
		"static inline void normalize_limbs(thread ulong *v, uint n) {\n"
		"  for (uint i = 0; i + 1 < n; i++) { v[i + 1] += v[i] >> 16; v[i] &= 0xffffUL; }\n"
		"}\n"
		"static inline void fe_reduce(thread uint *r, thread ulong *t) {\n"
		"  normalize_limbs(t, 32);\n"
		"  ulong acc[22];\n"
		"  for (uint i = 0; i < 22; i++) { acc[i] = 0; }\n"
		"  for (uint i = 0; i < 16; i++) { acc[i] = t[i]; }\n"
		"  for (uint i = 0; i < 16; i++) { ulong h = t[i + 16]; acc[i] += h * 977UL; acc[i + 2] += h; }\n"
		"  for (uint pass = 0; pass < 4; pass++) {\n"
		"    normalize_limbs(acc, 22);\n"
		"    for (uint k = 16; k < 22; k++) { ulong h = acc[k]; acc[k] = 0; acc[k - 16] += h * 977UL; acc[k - 14] += h; }\n"
		"  }\n"
		"  normalize_limbs(acc, 22);\n"
		"  for (uint i = 0; i < 16; i++) { r[i] = (uint)(acc[i] & 0xffffUL); }\n"
		"  for (uint i = 0; i < 4; i++) { if (fe_cmp_p(r) >= 0) { fe_sub_p(r); } }\n"
		"}\n"
		"static inline void fe_add(thread uint *r, thread const uint *a, thread const uint *b) {\n"
		"  ulong t[32];\n"
		"  for (uint i = 0; i < 32; i++) { t[i] = 0; }\n"
		"  for (uint i = 0; i < 16; i++) { t[i] = (ulong)a[i] + (ulong)b[i]; }\n"
		"  fe_reduce(r, t);\n"
		"}\n"
		"static inline void fe_sub(thread uint *r, thread const uint *a, thread const uint *b) {\n"
		"  uint borrow = 0;\n"
		"  for (uint i = 0; i < 16; i++) {\n"
		"    uint bi = b[i] + borrow;\n"
		"    if (a[i] >= bi) { r[i] = a[i] - bi; borrow = 0; } else { r[i] = (uint)(0x10000u + a[i] - bi); borrow = 1; }\n"
		"  }\n"
		"  if (borrow != 0) {\n"
		"    uint sub = 977;\n"
		"    for (uint i = 0; i < 16 && sub != 0; i++) { if (r[i] >= sub) { r[i] -= sub; sub = 0; } else { r[i] = (uint)(0x10000u + r[i] - sub); sub = 1; } }\n"
		"    sub = 1;\n"
		"    for (uint i = 2; i < 16 && sub != 0; i++) { if (r[i] >= sub) { r[i] -= sub; sub = 0; } else { r[i] = (uint)(0x10000u + r[i] - sub); sub = 1; } }\n"
		"  }\n"
		"}\n"
		"static inline void fe_mul_small(thread uint *r, thread const uint *a, uint m) {\n"
		"  ulong t[32];\n"
		"  for (uint i = 0; i < 32; i++) { t[i] = 0; }\n"
		"  for (uint i = 0; i < 16; i++) { t[i] = (ulong)a[i] * (ulong)m; }\n"
		"  fe_reduce(r, t);\n"
		"}\n"
		"static inline void fe_mul(thread uint *r, thread const uint *a, thread const uint *b) {\n"
		"  ulong t[32];\n"
		"  for (uint i = 0; i < 32; i++) { t[i] = 0; }\n"
		"  for (uint i = 0; i < 16; i++) { for (uint j = 0; j < 16; j++) { t[i + j] += (ulong)a[i] * (ulong)b[j]; } }\n"
		"  fe_reduce(r, t);\n"
		"}\n"
		"static inline void fe_sqr(thread uint *r, thread const uint *a) { fe_mul(r, a, a); }\n"
		"static inline bool exp_pm2_bit(uint bit) { return ((FE_PM2[bit >> 4] >> (bit & 15)) & 1u) != 0; }\n"
		"static inline void fe_inv(thread uint *r, thread const uint *a) {\n"
		"  uint result[16]; uint base[16]; uint tmp[16];\n"
		"  fe_one(result); fe_copy(base, a);\n"
		"  for (int bit = 255; bit >= 0; bit--) { fe_sqr(tmp, result); fe_copy(result, tmp); if (exp_pm2_bit((uint)bit)) { fe_mul(tmp, result, base); fe_copy(result, tmp); } }\n"
		"  fe_copy(r, result);\n"
		"}\n"
		"static inline void load_g(thread uint *gx, thread uint *gy) { for (uint i = 0; i < 16; i++) { gx[i] = SECP_GX[i]; gy[i] = SECP_GY[i]; } }\n"
		"static inline void point_set_g(thread uint *x, thread uint *y, thread uint *z, thread bool &inf) { load_g(x, y); fe_one(z); inf = false; }\n"
		"static inline void point_double(thread uint *x, thread uint *y, thread uint *z, thread bool &inf) {\n"
		"  if (inf || fe_is_zero(y)) { inf = true; return; }\n"
		"  uint A[16]; uint B[16]; uint C[16]; uint D[16]; uint E[16]; uint F[16]; uint T[16]; uint X3[16]; uint Y3[16]; uint Z3[16];\n"
		"  fe_sqr(A, x); fe_sqr(B, y); fe_sqr(C, B);\n"
		"  fe_add(T, x, B); fe_sqr(T, T); fe_sub(T, T, A); fe_sub(T, T, C); fe_mul_small(D, T, 2);\n"
		"  fe_mul_small(E, A, 3); fe_sqr(F, E); fe_mul_small(T, D, 2); fe_sub(X3, F, T);\n"
		"  fe_sub(T, D, X3); fe_mul(Y3, E, T); fe_mul_small(T, C, 8); fe_sub(Y3, Y3, T);\n"
		"  fe_mul(T, y, z); fe_mul_small(Z3, T, 2);\n"
		"  fe_copy(x, X3); fe_copy(y, Y3); fe_copy(z, Z3); inf = false;\n"
		"}\n"
		"static inline void point_add_g(thread uint *x, thread uint *y, thread uint *z, thread bool &inf) {\n"
		"  uint gx[16]; uint gy[16]; load_g(gx, gy);\n"
		"  if (inf) { fe_copy(x, gx); fe_copy(y, gy); fe_one(z); inf = false; return; }\n"
		"  uint Z1Z1[16]; uint U2[16]; uint S2[16]; uint H[16]; uint HH[16]; uint I[16]; uint J[16]; uint R[16]; uint V[16]; uint T[16]; uint X3[16]; uint Y3[16]; uint Z3[16];\n"
		"  fe_sqr(Z1Z1, z); fe_mul(U2, gx, Z1Z1); fe_mul(S2, z, Z1Z1); fe_mul(S2, gy, S2);\n"
		"  fe_sub(H, U2, x); fe_sub(R, S2, y);\n"
		"  if (fe_is_zero(H)) { if (fe_is_zero(R)) { point_double(x, y, z, inf); } else { inf = true; } return; }\n"
		"  fe_sqr(HH, H); fe_mul_small(I, HH, 4); fe_mul(J, H, I); fe_mul_small(R, R, 2); fe_mul(V, x, I);\n"
		"  fe_sqr(X3, R); fe_sub(X3, X3, J); fe_mul_small(T, V, 2); fe_sub(X3, X3, T);\n"
		"  fe_sub(T, V, X3); fe_mul(Y3, R, T); fe_mul(T, y, J); fe_mul_small(T, T, 2); fe_sub(Y3, Y3, T);\n"
		"  fe_add(T, z, H); fe_sqr(Z3, T); fe_sub(Z3, Z3, Z1Z1); fe_sub(Z3, Z3, HH);\n"
		"  fe_copy(x, X3); fe_copy(y, Y3); fe_copy(z, Z3); inf = false;\n"
		"}\n"
		"static inline bool scalar_bit(device const uchar *scalar, uint bit) { return ((scalar[31 - (bit >> 3)] >> (bit & 7)) & 1u) != 0; }\n"
		"static inline void store_fe_be(thread uchar *out, uint offset, thread const uint *a) { for (uint i = 0; i < 16; i++) { uint limb = a[15 - i]; out[offset + i * 2] = (uchar)(limb >> 8); out[offset + i * 2 + 1] = (uchar)(limb & 0xff); } }\n"
		"static inline void secp256k1_public_key(device const uchar *private_key, thread uchar *public_key) {\n"
		"  uint x[16]; uint y[16]; uint z[16]; bool inf = true;\n"
		"  for (int bit = 255; bit >= 0; bit--) { if (!inf) { point_double(x, y, z, inf); } if (scalar_bit(private_key, (uint)bit)) { point_add_g(x, y, z, inf); } }\n"
		"  if (inf) { for (uint i = 0; i < 64; i++) { public_key[i] = 0; } return; }\n"
		"  if (fe_is_one(z)) { store_fe_be(public_key, 0, x); store_fe_be(public_key, 32, y); return; }\n"
		"  uint zinv[16]; uint z2[16]; uint z3[16];\n"
		"  fe_inv(zinv, z); fe_sqr(z2, zinv); fe_mul(z3, x, z2); store_fe_be(public_key, 0, z3); fe_mul(z3, z2, zinv); fe_mul(zinv, y, z3); store_fe_be(public_key, 32, zinv);\n"
		"}\n"
		"static inline uchar state_byte(thread const ulong *state, uint byte_id) {\n"
		"  return (uchar)((state[byte_id >> 3] >> (8 * (byte_id & 7))) & 0xffUL);\n"
		"}\n"
		"static inline uchar address_nibble(thread const ulong *state, uint nibble_id) {\n"
		"  uchar value = state_byte(state, 12 + (nibble_id >> 1));\n"
		"  return (nibble_id & 1) == 0 ? ((value >> 4) & 0x0f) : (value & 0x0f);\n"
		"}\n"
		"static inline void keccakf(thread ulong *a) {\n"
		"  for (uint round = 0; round < 24; round++) {\n"
		"    ulong c[5];\n"
		"    ulong d[5];\n"
		"    ulong b[25];\n"
		"    for (uint x = 0; x < 5; x++) { c[x] = a[x] ^ a[x + 5] ^ a[x + 10] ^ a[x + 15] ^ a[x + 20]; }\n"
		"    for (uint x = 0; x < 5; x++) { d[x] = c[(x + 4) % 5] ^ rotl64(c[(x + 1) % 5], 1); }\n"
		"    for (uint x = 0; x < 5; x++) { for (uint y = 0; y < 5; y++) { a[x + 5 * y] ^= d[x]; } }\n"
		"    for (uint x = 0; x < 5; x++) {\n"
		"      for (uint y = 0; y < 5; y++) {\n"
		"        uint src = x + 5 * y;\n"
		"        uint dst = y + 5 * ((2 * x + 3 * y) % 5);\n"
		"        b[dst] = rotl64(a[src], KECCAKF_ROTC[src]);\n"
		"      }\n"
		"    }\n"
		"    for (uint y = 0; y < 5; y++) {\n"
		"      for (uint x = 0; x < 5; x++) { a[x + 5 * y] = b[x + 5 * y] ^ ((~b[((x + 1) % 5) + 5 * y]) & b[((x + 2) % 5) + 5 * y]); }\n"
		"    }\n"
		"    a[0] ^= KECCAKF_RNDC[round];\n"
		"  }\n"
		"}\n"
		"static inline void keccak256_public_key(thread const uchar *public_key, thread ulong *state) {\n"
		"  for (uint i = 0; i < 25; i++) { state[i] = 0; }\n"
		"  for (uint lane = 0; lane < 8; lane++) { state[lane] = load64_le_thread(public_key + lane * 8); }\n"
		"  state[8] ^= 0x0000000000000001UL;\n"
		"  state[16] ^= 0x8000000000000000UL;\n"
		"  keccakf(state);\n"
		"}\n"
		"kernel void count_matches(device const uchar *private_keys [[buffer(0)]], device const uchar *prefix [[buffer(1)]], device const uchar *suffix [[buffer(2)]], device atomic_uint *result [[buffer(3)]], constant uint &count [[buffer(4)]], constant uint &prefix_len [[buffer(5)]], constant uint &suffix_len [[buffer(6)]], uint id [[thread_position_in_grid]]) {\n"
		"  if (id >= count) { return; }\n"
		"  thread uchar public_key[64];\n"
		"  secp256k1_public_key(private_keys + id * 32, public_key);\n"
		"  ulong state[25];\n"
		"  keccak256_public_key(public_key, state);\n"
		"  bool ok = true;\n"
		"  for (uint i = 0; i < prefix_len; i++) {\n"
		"    if (address_nibble(state, i) != prefix[i]) { ok = false; }\n"
		"  }\n"
		"  for (uint i = 0; i < suffix_len; i++) {\n"
		"    if (address_nibble(state, 40 - suffix_len + i) != suffix[i]) { ok = false; }\n"
		"  }\n"
		"  if (ok) { atomic_fetch_add_explicit(result, 1u, memory_order_relaxed); }\n"
		"}\n";
}

static void* metal_create_match_context(char **error_msg) {
	@autoreleasepool {
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) {
			*error_msg = strdup("metal device not available");
			return NULL;
		}

		NSError *error = nil;
		id<MTLLibrary> library = [device newLibraryWithSource:metal_match_source() options:nil error:&error];
		if (library == nil) {
			*error_msg = metal_copy_error(error);
			return NULL;
		}

		id<MTLFunction> function = [library newFunctionWithName:@"count_matches"];
		if (function == nil) {
			*error_msg = strdup("metal function count_matches not found");
			return NULL;
		}
		id<MTLComputePipelineState> pipeline = [device newComputePipelineStateWithFunction:function error:&error];
		if (pipeline == nil) {
			*error_msg = metal_copy_error(error);
			return NULL;
		}

		id<MTLCommandQueue> queue = [device newCommandQueue];
		if (queue == nil) {
			*error_msg = strdup("failed to create metal command queue");
			return NULL;
		}

		BlocoMetalMatchContext *context = [BlocoMetalMatchContext new];
		context.device = device;
		context.pipeline = pipeline;
		context.queue = queue;
		return (__bridge_retained void *)context;
	}
}

static void metal_release_match_context(void *context_ptr) {
	if (context_ptr != NULL) {
		CFRelease(context_ptr);
	}
}

static char* metal_context_device_name(void *context_ptr) {
	@autoreleasepool {
		if (context_ptr == NULL) {
			return NULL;
		}
		BlocoMetalMatchContext *context = (__bridge BlocoMetalMatchContext *)context_ptr;
		return metal_copy_string([context.device name]);
	}
}

static int metal_run_match(void *context_ptr, const uint8_t *private_keys, uint32_t count, const uint8_t *prefix, uint32_t prefix_len, const uint8_t *suffix, uint32_t suffix_len, uint32_t *matches, double *buffer_ns, double *kernel_ns, char **error_msg) {
	@autoreleasepool {
		if (context_ptr == NULL) {
			*error_msg = strdup("metal context not initialized");
			return 1;
		}
		BlocoMetalMatchContext *context = (__bridge BlocoMetalMatchContext *)context_ptr;
		id<MTLDevice> device = context.device;
		id<MTLComputePipelineState> pipeline = context.pipeline;
		id<MTLCommandQueue> queue = context.queue;

		CFAbsoluteTime buffer_start = CFAbsoluteTimeGetCurrent();
		NSUInteger private_key_byte_count = (NSUInteger)count * 32;
		id<MTLBuffer> private_key_buffer = [device newBufferWithLength:private_key_byte_count options:MTLResourceStorageModeShared];
		NSUInteger prefix_byte_count = prefix_len == 0 ? 1 : prefix_len;
		id<MTLBuffer> prefix_buffer = [device newBufferWithLength:prefix_byte_count options:MTLResourceStorageModeShared];
		NSUInteger suffix_byte_count = suffix_len == 0 ? 1 : suffix_len;
		id<MTLBuffer> suffix_buffer = [device newBufferWithLength:suffix_byte_count options:MTLResourceStorageModeShared];
		id<MTLBuffer> result_buffer = [device newBufferWithLength:sizeof(uint32_t) options:MTLResourceStorageModeShared];
		if (private_key_buffer == nil || prefix_buffer == nil || suffix_buffer == nil || result_buffer == nil) {
			*error_msg = strdup("failed to allocate metal buffers");
			return 5;
		}

		memcpy([private_key_buffer contents], private_keys, private_key_byte_count);
		if (prefix_len > 0) {
			memcpy([prefix_buffer contents], prefix, prefix_len);
		}
		if (suffix_len > 0) {
			memcpy([suffix_buffer contents], suffix, suffix_len);
		}
		uint32_t *result_contents = (uint32_t *)[result_buffer contents];
		*result_contents = 0;
		CFAbsoluteTime buffer_end = CFAbsoluteTimeGetCurrent();

		id<MTLCommandBuffer> command_buffer = [queue commandBuffer];
		id<MTLComputeCommandEncoder> encoder = [command_buffer computeCommandEncoder];
		if (command_buffer == nil || encoder == nil) {
			*error_msg = strdup("failed to create metal command encoder");
			return 7;
		}

		[encoder setComputePipelineState:pipeline];
		[encoder setBuffer:private_key_buffer offset:0 atIndex:0];
		[encoder setBuffer:prefix_buffer offset:0 atIndex:1];
		[encoder setBuffer:suffix_buffer offset:0 atIndex:2];
		[encoder setBuffer:result_buffer offset:0 atIndex:3];
		[encoder setBytes:&count length:sizeof(count) atIndex:4];
		[encoder setBytes:&prefix_len length:sizeof(prefix_len) atIndex:5];
		[encoder setBytes:&suffix_len length:sizeof(suffix_len) atIndex:6];

		NSUInteger width = (NSUInteger)count;
		NSUInteger threadgroup_width = pipeline.maxTotalThreadsPerThreadgroup;
		if (threadgroup_width > width) {
			threadgroup_width = width;
		}
		MTLSize grid_size = MTLSizeMake(width, 1, 1);
		MTLSize threadgroup_size = MTLSizeMake(threadgroup_width, 1, 1);

		CFAbsoluteTime start = CFAbsoluteTimeGetCurrent();
		[encoder dispatchThreads:grid_size threadsPerThreadgroup:threadgroup_size];
		[encoder endEncoding];
		[command_buffer commit];
		[command_buffer waitUntilCompleted];
		CFAbsoluteTime end = CFAbsoluteTimeGetCurrent();

		if ([command_buffer status] == MTLCommandBufferStatusError) {
			*error_msg = metal_copy_error([command_buffer error]);
			return 8;
		}

		*matches = *result_contents;
		*buffer_ns = (buffer_end - buffer_start) * 1000000000.0;
		*kernel_ns = (end - start) * 1000000000.0;
		return 0;
	}
}

*/
import "C"

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
	"unsafe"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"bloco-vgen/pkg/wallet"
)

type MetalEngine struct {
	deviceName string
	context    unsafe.Pointer
}

func MetalAvailable() bool {
	return C.metal_available() == 1
}

func MetalUnavailableReason() string {
	return "metal device not available; using cpu"
}

func NewMetalEngine() (BenchmarkEngine, error) {
	if !MetalAvailable() {
		return nil, fmt.Errorf("metal engine is not available yet; use --engine cpu or --engine auto")
	}

	context, err := newMetalMatchContext()
	if err != nil {
		return nil, err
	}

	name, err := metalContextDeviceName(context)
	if err != nil {
		C.metal_release_match_context(context)
		return nil, err
	}

	engine := &MetalEngine{deviceName: name, context: context}
	runtime.SetFinalizer(engine, (*MetalEngine).Close)
	return engine, nil
}

func (e *MetalEngine) Close() {
	if e.context != nil {
		C.metal_release_match_context(e.context)
		e.context = nil
		runtime.SetFinalizer(e, nil)
	}
}

func (e *MetalEngine) Name() string {
	return NameMetal
}

func (e *MetalEngine) RunBenchmark(ctx context.Context, options BenchmarkOptions, sampleInterval time.Duration, onSample func(Sample)) (*wallet.BenchmarkResult, error) {
	benchmarkCtx := ctx
	cancel := func() {}
	if options.Duration > 0 {
		benchmarkCtx, cancel = context.WithTimeout(ctx, options.Duration)
	}
	defer cancel()

	targetAttempts := options.Attempts
	if targetAttempts <= 0 {
		targetAttempts = 1
	}
	prefixNibbles, err := patternNibbles(options.Criteria.Prefix)
	if err != nil {
		return nil, err
	}
	suffixNibbles, err := patternNibbles(options.Criteria.Suffix)
	if err != nil {
		return nil, err
	}

	batchSize := options.BatchSize
	if batchSize <= 0 || batchSize > targetAttempts {
		batchSize = targetAttempts
	}

	start := time.Now()
	lastSample := start
	lastSampleAttempts := int64(0)
	totalAttempts := int64(0)
	totalMatches := int64(0)
	stages := stageTotals{}
	metalBufferDuration := time.Duration(0)
	kernelDuration := time.Duration(0)
	var speedSamples []float64
	var durationSamples []time.Duration

	for totalAttempts < int64(targetAttempts) {
		select {
		case <-benchmarkCtx.Done():
			if totalAttempts == 0 {
				return nil, benchmarkCtx.Err()
			}
			goto complete
		default:
		}

		currentBatchSize := min(batchSize, targetAttempts-int(totalAttempts))
		batchAttempts, batchMatches, batchStages, batchMetalBufferDuration, batchKernelDuration, err := runMetalHybridBatch(e.context, benchmarkCtx, currentBatchSize, prefixNibbles, suffixNibbles)
		if err != nil {
			if totalAttempts > 0 && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
				break
			}
			return nil, err
		}

		totalAttempts += int64(batchAttempts)
		totalMatches += int64(batchMatches)
		stages.Entropy += batchStages.Entropy
		stages.Scalar += batchStages.Scalar
		stages.Hash += batchStages.Hash
		stages.Match += batchStages.Match
		metalBufferDuration += batchMetalBufferDuration
		kernelDuration += batchKernelDuration

		now := time.Now()
		if onSample != nil && sampleInterval > 0 && now.Sub(lastSample) >= sampleInterval {
			elapsed := now.Sub(lastSample)
			speed := float64(totalAttempts-lastSampleAttempts) / elapsed.Seconds()
			speedSamples = append(speedSamples, speed)
			durationSamples = append(durationSamples, elapsed)
			onSample(Sample{Attempts: totalAttempts, Matches: totalMatches, Speed: speed, Elapsed: now.Sub(start)})
			lastSample = now
			lastSampleAttempts = totalAttempts
		}
	}

complete:
	totalDuration := time.Since(start)
	if totalDuration <= 0 {
		totalDuration = kernelDuration
	}
	result := e.buildHybridBenchmarkResult(options, batchSize, totalAttempts, totalMatches, totalDuration, speedSamples, durationSamples, stages, metalBufferDuration, kernelDuration)

	if onSample != nil {
		onSample(Sample{
			Attempts: result.TotalAttempts,
			Matches:  result.Matches,
			Speed:    result.AverageSpeed,
			Elapsed:  result.TotalDuration,
			Result:   result,
		})
	}

	return result, nil
}

func (e *MetalEngine) buildHybridBenchmarkResult(options BenchmarkOptions, batchSize int, totalAttempts int64, totalMatches int64, totalDuration time.Duration, speedSamples []float64, durationSamples []time.Duration, stages stageTotals, metalBufferDuration time.Duration, kernelDuration time.Duration) *wallet.BenchmarkResult {
	averageSpeed := 0.0
	if totalDuration > 0 {
		averageSpeed = float64(totalAttempts) / totalDuration.Seconds()
	}
	if len(speedSamples) == 0 && averageSpeed > 0 {
		speedSamples = []float64{averageSpeed}
		durationSamples = []time.Duration{totalDuration}
	}
	minSpeed, maxSpeed := speedRange(speedSamples)

	return &wallet.BenchmarkResult{
		Engine:                 NameMetal,
		RequestedEngine:        options.RequestedEngine,
		DeviceName:             e.deviceName,
		Network:                options.Network,
		Pattern:                options.Criteria.GetPattern(),
		BatchSize:              batchSize,
		IsHybrid:               true,
		Matches:                totalMatches,
		TotalAttempts:          totalAttempts,
		TotalDuration:          totalDuration,
		AverageSpeed:           averageSpeed,
		MinSpeed:               minSpeed,
		MaxSpeed:               maxSpeed,
		SpeedSamples:           speedSamples,
		DurationSamples:        durationSamples,
		SingleThreadSpeed:      averageSpeed,
		ThreadCount:            1,
		ScalabilityEfficiency:  1,
		ThreadBalanceScore:     1,
		ThreadUtilization:      1,
		SpeedupVsSingleThread:  1,
		EntropyDuration:        stages.Entropy,
		ScalarBaseMultDuration: stages.Scalar,
		HashFormatDuration:     stages.Hash,
		MatchDuration:          stages.Match,
		MetalBufferDuration:    metalBufferDuration,
		KernelDuration:         kernelDuration,
	}
}

func newMetalMatchContext() (unsafe.Pointer, error) {
	var errorMsg *C.char
	context := C.metal_create_match_context(&errorMsg)
	if errorMsg != nil {
		defer C.free(unsafe.Pointer(errorMsg))
	}
	if context == nil {
		if errorMsg != nil {
			return nil, fmt.Errorf("failed to initialize metal context: %s", C.GoString(errorMsg))
		}
		return nil, fmt.Errorf("failed to initialize metal context")
	}
	return context, nil
}

func metalContextDeviceName(context unsafe.Pointer) (string, error) {
	name := C.metal_context_device_name(context)
	if name == nil {
		return "", fmt.Errorf("metal device not available")
	}
	defer C.free(unsafe.Pointer(name))
	return C.GoString(name), nil
}

func generateEthereumPrivateKeyBatch(ctx context.Context, attempts int) ([]byte, stageTotals, error) {
	if attempts <= 0 {
		attempts = 1
	}

	privateKeys := make([]byte, attempts*32)
	totals := stageTotals{}
	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			return nil, totals, ctx.Err()
		default:
		}

		privateKey, entropyDuration, err := generateEthereumPrivateKeyAttempt()
		if err != nil {
			return nil, totals, err
		}
		copy(privateKeys[i*32:(i+1)*32], privateKey[:])
		totals.Entropy += entropyDuration
	}

	return privateKeys, totals, nil
}

func runMetalHybridBatch(context unsafe.Pointer, ctx context.Context, attempts int, prefix []byte, suffix []byte) (int, uint32, stageTotals, time.Duration, time.Duration, error) {
	privateKeys, stages, err := generateEthereumPrivateKeyBatch(ctx, attempts)
	if err != nil {
		return 0, 0, stages, 0, 0, err
	}

	cpuMatches, scalarDuration, hashDuration, matchDuration := countPrivateKeyMatches(privateKeys, prefix, suffix)
	stages.Scalar = scalarDuration
	stages.Hash = hashDuration
	stages.Match = matchDuration

	gpuMatches, metalBufferDuration, kernelDuration, err := runMetalMatch(context, privateKeys, prefix, suffix)
	if err != nil {
		return 0, 0, stages, metalBufferDuration, kernelDuration, err
	}
	if gpuMatches != cpuMatches {
		return 0, 0, stages, metalBufferDuration, kernelDuration, fmt.Errorf("metal match validation failed: gpu=%d cpu=%d", gpuMatches, cpuMatches)
	}

	return len(privateKeys) / 32, gpuMatches, stages, metalBufferDuration, kernelDuration, nil
}

func countPrivateKeyMatches(privateKeys []byte, prefix []byte, suffix []byte) (uint32, time.Duration, time.Duration, time.Duration) {
	count := len(privateKeys) / 32
	hasher := sha3.NewLegacyKeccak256()
	matches := uint32(0)
	scalarDuration := time.Duration(0)
	hashDuration := time.Duration(0)
	matchDuration := time.Duration(0)

	for i := 0; i < count; i++ {
		privateKey := privateKeys[i*32 : (i+1)*32]
		stageStart := time.Now()
		x, y := ethcrypto.S256().ScalarBaseMult(privateKey)
		scalarDuration += time.Since(stageStart)

		var publicKey [64]byte
		x.FillBytes(publicKey[:32])
		y.FillBytes(publicKey[32:])

		stageStart = time.Now()
		address := EthereumAddressBytesFromPublicKey(publicKey[:], hasher)
		hashDuration += time.Since(stageStart)

		stageStart = time.Now()
		if addressMatches(address[:], prefix, suffix) {
			matches++
		}
		matchDuration += time.Since(stageStart)
	}

	return matches, scalarDuration, hashDuration, matchDuration
}

func runMetalMatch(context unsafe.Pointer, privateKeys []byte, prefix []byte, suffix []byte) (uint32, time.Duration, time.Duration, error) {
	count, err := metalPrivateKeyBatchCount(privateKeys, maxMetalBatchCandidates())
	if err != nil {
		return 0, 0, 0, err
	}

	var bufferNS C.double
	var kernelNS C.double
	var errorMsg *C.char
	var matches C.uint32_t
	privateKeyPtr := (*C.uint8_t)(unsafe.Pointer(&privateKeys[0]))
	prefixPtr := cBytePointer(prefix)
	suffixPtr := cBytePointer(suffix)
	code := C.metal_run_match(context, privateKeyPtr, C.uint32_t(count), prefixPtr, C.uint32_t(len(prefix)), suffixPtr, C.uint32_t(len(suffix)), &matches, &bufferNS, &kernelNS, &errorMsg)
	if errorMsg != nil {
		defer C.free(unsafe.Pointer(errorMsg))
	}
	if code != 0 {
		if errorMsg != nil {
			return 0, 0, 0, fmt.Errorf("metal match kernel failed: %s", C.GoString(errorMsg))
		}
		return 0, 0, 0, fmt.Errorf("metal match kernel failed with code %d", int(code))
	}

	return uint32(matches), durationFromNanoseconds(float64(bufferNS)), durationFromNanoseconds(float64(kernelNS)), nil
}

func metalPrivateKeyBatchCount(privateKeys []byte, maxCount int) (uint32, error) {
	if len(privateKeys) == 0 {
		return 0, fmt.Errorf("metal match requires at least one private key")
	}
	if len(privateKeys)%32 != 0 {
		return 0, fmt.Errorf("metal match private key buffer must be a multiple of 32 bytes, got %d", len(privateKeys))
	}
	count := len(privateKeys) / 32
	if maxCount <= 0 {
		return 0, fmt.Errorf("metal match maximum batch size must be positive, got %d", maxCount)
	}
	if count > maxCount {
		return 0, fmt.Errorf("metal match batch too large: %d private keys exceeds maximum %d", count, maxCount)
	}
	return uint32(count), nil
}

func maxMetalBatchCandidates() int {
	return runtime.NumCPU() * 1000000
}

func durationFromNanoseconds(value float64) time.Duration {
	if value <= 0 {
		return 0
	}
	duration := time.Duration(value)
	if duration <= 0 {
		return time.Nanosecond
	}
	return duration
}

func cBytePointer(values []byte) *C.uint8_t {
	if len(values) == 0 {
		return nil
	}
	return (*C.uint8_t)(unsafe.Pointer(&values[0]))
}
