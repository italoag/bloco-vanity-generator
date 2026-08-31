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

static void metal_zero_buffer(id<MTLBuffer> buffer) {
	if (buffer != nil) {
		memset([buffer contents], 0, [buffer length]);
	}
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
@property(nonatomic, strong) id<MTLBuffer> private_key_buffer;
@property(nonatomic, strong) id<MTLBuffer> prefix_buffer;
@property(nonatomic, strong) id<MTLBuffer> suffix_buffer;
@property(nonatomic, strong) id<MTLBuffer> match_index_buffer;
@property(nonatomic, strong) id<MTLBuffer> match_count_buffer;
@property(nonatomic, strong) id<MTLBuffer> comb_table_buffer;
@property(nonatomic, assign) uint32_t private_key_capacity;
@property(nonatomic, assign) uint32_t prefix_capacity;
@property(nonatomic, assign) uint32_t suffix_capacity;
@property(nonatomic, assign) uint32_t match_index_capacity;
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
		// ---- 4x64 field arithmetic for secp256k1 (p = 2^256 - 2^32 - 977) ----
		"static inline ulong load64_le_thread(thread const uchar *bytes) {\n"
		"  ulong value = 0;\n"
		"  for (uint i = 0; i < 8; i++) { value |= ((ulong)bytes[i]) << (8 * i); }\n"
		"  return value;\n"
		"}\n"
		"constant ulong SECP_P0 = 0xFFFFFFFEFFFFFC2FUL;\n"
		"constant ulong SECP_P1 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"constant ulong SECP_P2 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"constant ulong SECP_P3 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"constant ulong SECP_C977 = 0x3D1UL;\n"
		"constant ulong SECP_PM2_0 = 0xFFFFFFFEFFFFFC2DUL;\n"
		"constant ulong SECP_PM2_1 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"constant ulong SECP_PM2_2 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"constant ulong SECP_PM2_3 = 0xFFFFFFFFFFFFFFFFUL;\n"
		"static inline void fe_zero(thread ulong *r) { r[0] = 0; r[1] = 0; r[2] = 0; r[3] = 0; }\n"
		"static inline void fe_one(thread ulong *r) { fe_zero(r); r[0] = 1; }\n"
		"static inline void fe_copy(thread ulong *r, thread const ulong *a) { r[0] = a[0]; r[1] = a[1]; r[2] = a[2]; r[3] = a[3]; }\n"
		"static inline bool fe_is_zero(thread const ulong *a) { return a[0] == 0 && a[1] == 0 && a[2] == 0 && a[3] == 0; }\n"
		"static inline bool fe_is_one(thread const ulong *a) { return a[0] == 1 && a[1] == 0 && a[2] == 0 && a[3] == 0; }\n"
		// Addition mod p: exact (a + b < 2p, so a single 2^256 wrap is fixed
		// by adding back c = 2^32 + 977).
		"static inline void fe_add(thread ulong *r, thread const ulong *a, thread const ulong *b) {\n"
		"  ulong s0 = a[0] + b[0]; ulong c = (s0 < a[0]) ? 1 : 0;\n"
		"  ulong s1 = a[1] + b[1]; ulong c1 = (s1 < a[1]) ? 1 : 0; s1 += c; c1 += (s1 < c) ? 1 : 0;\n"
		"  ulong s2 = a[2] + b[2]; ulong c2 = (s2 < a[2]) ? 1 : 0; s2 += c1; c2 += (s2 < c1) ? 1 : 0;\n"
		"  ulong s3 = a[3] + b[3]; ulong w = (s3 < a[3]) ? 1 : 0; s3 += c2; w += (s3 < c2) ? 1 : 0;\n"
		"  if (w != 0) {\n"
		"    // wrapped past 2^256: add back c = 2^32 + 977 (2^256 mod p).\n"
		"    // Both terms fit in limb 0; the final carry is provably zero\n"
		"    // because a + b < 2p, so s + c = a + b - p < p < 2^256.\n"
		"    s0 += SECP_C977; ulong c0 = (s0 < SECP_C977) ? 1 : 0;\n"
		"    s0 += 0x100000000UL; c0 += (s0 < 0x100000000UL) ? 1 : 0;\n"
		"    ulong s1b = s1 + c0; ulong c1b = (s1b < c0) ? 1 : 0; s1 = s1b;\n"
		"    ulong s2b = s2 + c1b; ulong c2b = (s2b < c1b) ? 1 : 0; s2 = s2b;\n"
		"    s3 += c2b;\n"
		"  }\n"
		"  r[0] = s0; r[1] = s1; r[2] = s2; r[3] = s3;\n"
		"}\n"
		// Subtraction mod p: exact (adds p back when a < b).
		"static inline void fe_sub(thread ulong *r, thread const ulong *a, thread const ulong *b) {\n"
		"  ulong d0 = a[0] - b[0]; bool br = a[0] < b[0];\n"
		"  ulong d1 = a[1] - b[1] - (br ? 1 : 0); bool br1 = br ? (a[1] <= b[1]) : (a[1] < b[1]);\n"
		"  ulong d2 = a[2] - b[2] - (br1 ? 1 : 0); bool br2 = br1 ? (a[2] <= b[2]) : (a[2] < b[2]);\n"
		"  ulong d3 = a[3] - b[3] - (br2 ? 1 : 0); bool br3 = br2 ? (a[3] <= b[3]) : (a[3] < b[3]);\n"
		"  if (br3) {\n"
		"    d0 += SECP_P0; ulong c = (d0 < SECP_P0) ? 1 : 0;\n"
		"    ulong t1 = d1 + SECP_P1; ulong c1 = (t1 < SECP_P1) ? 1 : 0; t1 += c; c1 += (t1 < c) ? 1 : 0; d1 = t1;\n"
		"    ulong t2 = d2 + SECP_P2; ulong c2 = (t2 < SECP_P2) ? 1 : 0; t2 += c1; c2 += (t2 < c1) ? 1 : 0; d2 = t2;\n"
		"    d3 += SECP_P3 + c2;\n"
		"  }\n"
		"  r[0] = d0; r[1] = d1; r[2] = d2; r[3] = d3;\n"
		"}\n"
		// Modular reduction of an 8-limb product into 4 limbs, folding the
		// top limbs with c = 2^32 + 977 until only a tiny overflow remains,
		// then conditional subtraction of p.
		"static inline void fe_reduce(thread ulong *r, thread const ulong *t) {\n"
		"  ulong a0 = t[0]; ulong a1 = t[1]; ulong a2 = t[2]; ulong a3 = t[3];\n"
		"  ulong ov0 = t[4]; ulong ov1 = t[5]; ulong ov2 = t[6]; ulong ov3 = t[7];\n"
		"  // Fold pass 1: full 4-limb overflow * c, exact into a[0..4].\n"
		"  ulong a4 = 0;\n"
		"  for (uint k = 0; k < 4; k++) {\n"
		"    ulong x = (k == 0) ? ov0 : (k == 1) ? ov1 : (k == 2) ? ov2 : ov3;\n"
		"    ulong mlo = x * SECP_C977;\n"
		"    ulong mhi = mulhi(x, SECP_C977);\n"
		"    ulong sh = x << 32;\n"
		"    ulong shh = x >> 32;\n"
		"    ulong v = (k == 0) ? a0 : (k == 1) ? a1 : (k == 2) ? a2 : a3;\n"
		"    ulong s = v + mlo; ulong c = (s < mlo) ? 1 : 0; v = s;\n"
		"    s = v + sh; c += (s < sh) ? 1 : 0; v = s;\n"
		"    if (k == 0) { a0 = v; } else if (k == 1) { a1 = v; } else if (k == 2) { a2 = v; } else { a3 = v; }\n"
		"    ulong w = (k == 0) ? a1 : (k == 1) ? a2 : (k == 2) ? a3 : a4;\n"
		"    ulong s1 = w + mhi; ulong c1 = (s1 < mhi) ? 1 : 0; w = s1;\n"
		"    s1 = w + shh; c1 += (s1 < shh) ? 1 : 0; w = s1;\n"
		"    s1 = w + c; c1 += (s1 < c) ? 1 : 0; w = s1;\n"
		"    if (k == 0) { a1 = w; } else if (k == 1) { a2 = w; } else if (k == 2) { a3 = w; } else { a4 = w; }\n"
		"    if (c1 != 0) {\n"
		"      if (k == 0) { a2 += c1; if (a2 < c1) { a3 += 1; if (a3 == 0) { a4 += 1; } } }\n"
		"      else if (k == 1) { a3 += c1; if (a3 < c1) { a4 += 1; } }\n"
		"      else if (k == 2) { a4 += c1; }\n"
		"      // k == 3: c1 would land in a[5]; the total is < 2^289 so it is zero.\n"
		"    }\n"
		"  }\n"
		"  // Fold passes 2-4: 1-limb overflow (a4) * c with full carry capture.\n"
		"  for (uint pass = 1; pass < 4; pass++) {\n"
		"    if (a4 == 0) { break; }\n"
		"    ulong x = a4;\n"
		"    ulong mlo = x * SECP_C977;\n"
		"    ulong mhi = mulhi(x, SECP_C977);\n"
		"    ulong sh = x << 32;\n"
		"    ulong shh = x >> 32;\n"
		"    ulong s = a0 + mlo; ulong c = (s < mlo) ? 1 : 0; a0 = s;\n"
		"    s = a0 + sh; c += (s < sh) ? 1 : 0; a0 = s;\n"
		"    ulong s1 = a1 + mhi; ulong c1 = (s1 < mhi) ? 1 : 0; s1 += shh; c1 += (s1 < shh) ? 1 : 0; s1 += c; c1 += (s1 < c) ? 1 : 0; a1 = s1;\n"
		"    ulong s2 = a2 + c1; ulong c2 = (s2 < c1) ? 1 : 0; a2 = s2;\n"
		"    ulong s3 = a3 + c2; ulong c3 = (s3 < c2) ? 1 : 0; a3 = s3;\n"
		"    a4 = c3;\n"
		"  }\n"
		"  // Value is now < 2^256 + c (exact mod p); normalize with conditional\n"
		"  // subtractions of p (two passes cover the residual excess).\n"
		"  for (uint pass = 0; pass < 2; pass++) {\n"
		"    bool ge = a3 > SECP_P3;\n"
		"    if (!ge && a3 == SECP_P3) {\n"
		"      if (a2 > SECP_P2) { ge = true; }\n"
		"      else if (a2 == SECP_P2) {\n"
		"        if (a1 > SECP_P1) { ge = true; }\n"
		"        else if (a1 == SECP_P1) { ge = a0 >= SECP_P0; }\n"
		"      }\n"
		"    }\n"
		"    if (!ge) { break; }\n"
		"    ulong d0 = a0 - SECP_P0; bool br = a0 < SECP_P0;\n"
		"    ulong d1 = a1 + 1 - (br ? 1 : 0); bool br1 = (a1 != SECP_P1) || (br && a1 == SECP_P1);\n"
		"    ulong d2 = a2 + 1 - (br1 ? 1 : 0); bool br2 = (a2 != SECP_P2) || (br1 && a2 == SECP_P2);\n"
		"    ulong d3 = a3 + 1 - (br2 ? 1 : 0);\n"
		"    a0 = d0; a1 = d1; a2 = d2; a3 = d3;\n"
		"  }\n"
		"  r[0] = a0; r[1] = a1; r[2] = a2; r[3] = a3;\n"
		"}\n"
		// Schoolbook 4x4 multiplication into 8 limbs with full carry
		// propagation. The total product is < 2^512, so the carry chain from
		// t[k+2] can only reach the top of the array.
		"static inline void fe_mul(thread ulong *r, thread const ulong *a, thread const ulong *b) {\n"
		"  ulong t[8];\n"
		"  for (uint i = 0; i < 8; i++) { t[i] = 0; }\n"
		"  for (uint i = 0; i < 4; i++) {\n"
		"    for (uint j = 0; j < 4; j++) {\n"
		"      ulong mlo = a[i] * b[j];\n"
		"      ulong mhi = mulhi(a[i], b[j]);\n"
		"      uint k = i + j;\n"
		"      ulong s = t[k] + mlo; ulong c = (s < mlo) ? 1 : 0; t[k] = s;\n"
		"      ulong s1 = t[k + 1] + mhi; ulong c1 = (s1 < t[k + 1]) ? 1 : 0; s1 += c; c1 += (s1 < c) ? 1 : 0; t[k + 1] = s1;\n"
		"      if (k + 2 < 8) {\n"
		"        ulong carry = c1;\n"
		"        uint idx = k + 2;\n"
		"        while (carry != 0 && idx < 8) {\n"
		"          t[idx] += carry;\n"
		"          carry = (t[idx] < carry) ? 1 : 0;\n"
		"          idx++;\n"
		"        }\n"
		"      }\n"
		"    }\n"
		"  }\n"
		"  fe_reduce(r, t);\n"
		"}\n"
		"static inline void fe_sqr(thread ulong *r, thread const ulong *a) { fe_mul(r, a, a); }\n"
		// Fermat inversion via exponentiation by p - 2.
		"static inline void fe_inv(thread ulong *r, thread const ulong *a) {\n"
		"  ulong result[4]; ulong base[4]; ulong tmp[4];\n"
		"  fe_one(result); fe_copy(base, a);\n"
		"  for (uint bit = 256; bit-- > 0; ) {\n"
		"    uint b = bit;\n"
		"    fe_sqr(tmp, result); fe_copy(result, tmp);\n"
		"    uint limb = b >> 6;\n"
		"    ulong pm2 = (limb == 0) ? SECP_PM2_0 : (limb == 1) ? SECP_PM2_1 : (limb == 2) ? SECP_PM2_2 : SECP_PM2_3;\n"
		"    if (((pm2 >> (b & 63)) & 1UL) != 0) { fe_mul(tmp, result, base); fe_copy(result, tmp); }\n"
		"  }\n"
		"  fe_copy(r, result);\n"
		"}\n"
		// Jacobian point addition (add-2007-bl). Inputs may alias the output.
		"static inline void jacobian_add(thread ulong *x3, thread ulong *y3, thread ulong *z3, thread const ulong *x1, thread const ulong *y1, thread const ulong *z1, thread const ulong *x2, thread const ulong *y2, thread const ulong *z2) {\n"
		"  if (fe_is_zero(z1)) { fe_copy(x3, x2); fe_copy(y3, y2); fe_copy(z3, z2); return; }\n"
		"  if (fe_is_zero(z2)) { fe_copy(x3, x1); fe_copy(y3, y1); fe_copy(z3, z1); return; }\n"
		"  ulong z1z1[4]; ulong z2z2[4]; ulong u1[4]; ulong u2[4]; ulong s1[4]; ulong s2[4];\n"
		"  ulong h[4]; ulong ii[4]; ulong jj[4]; ulong rr[4]; ulong v[4]; ulong t[4];\n"
		"  ulong nx[4]; ulong ny[4]; ulong nz[4];\n"
		"  fe_sqr(z1z1, z1); fe_sqr(z2z2, z2);\n"
		"  fe_mul(u1, x1, z2z2); fe_mul(u2, x2, z1z1);\n"
		"  fe_mul(s1, y1, z2); fe_mul(s1, s1, z2z2);\n"
		"  fe_mul(s2, y2, z1); fe_mul(s2, s2, z1z1);\n"
		"  fe_sub(h, u2, u1);\n"
		"  if (fe_is_zero(h)) {\n"
		"    fe_sub(t, s2, s1);\n"
		"    if (fe_is_zero(t)) {\n"
		"      // P == Q: point doubling.\n"
		"      ulong a[4]; ulong b[4]; ulong c[4]; ulong d[4]; ulong e[4]; ulong f[4];\n"
		"      fe_sqr(a, x1); fe_sqr(b, y1); fe_sqr(c, b);\n"
		"      fe_add(t, x1, b); fe_sqr(t, t); fe_sub(t, t, a); fe_sub(t, t, c); fe_add(d, t, t);\n"
		"      fe_add(t, a, a); fe_add(e, t, a); fe_sqr(f, e);\n"
		"      fe_add(t, d, d); fe_sub(nx, f, t);\n"
		"      fe_sub(t, d, nx); fe_mul(ny, e, t); fe_add(t, c, c); fe_add(t, t, t); fe_add(t, t, t); fe_sub(ny, ny, t);\n"
		"      fe_mul(t, y1, z1); fe_add(nz, t, t);\n"
		"    } else {\n"
		"      fe_zero(x3); fe_zero(y3); fe_zero(z3);\n"
		"      return;\n"
		"    }\n"
		"  } else {\n"
		"    fe_add(t, h, h); fe_sqr(ii, t);\n"
		"    fe_mul(jj, h, ii);\n"
		"    fe_sub(t, s2, s1); fe_add(rr, t, t);\n"
		"    fe_mul(v, u1, ii);\n"
		"    fe_sqr(t, rr); fe_sub(nx, t, jj); fe_add(t, v, v); fe_sub(nx, nx, t);\n"
		"    fe_sub(t, v, nx); fe_mul(t, rr, t); fe_mul(ny, s1, jj); fe_add(ny, ny, ny); fe_sub(ny, t, ny);\n"
		"    fe_add(t, z1, z2); fe_sqr(t, t); fe_sub(t, t, z1z1); fe_sub(t, t, z2z2); fe_mul(nz, t, h);\n"
		"  }\n"
		"  fe_copy(x3, nx); fe_copy(y3, ny); fe_copy(z3, nz);\n"
		"}\n"
		// Fixed-base comb: table[window 0..63][entry 0..15] holds the Jacobian
		// point (entry * 2^(4*window)) * G with Z = 1 (entry 0 is infinity).
		"constant uint COMB_WINDOWS = 64;\n"
		"constant uint COMB_ENTRIES = 16;\n"
		"static inline uint scalar_window_thread(thread const uchar *scalar, uint w) {\n"
		"  uint bit = w * 4;\n"
		"  uint byte_id = 31 - (bit >> 3);\n"
		"  uint shift = bit & 7;\n"
		"  uint value = (uint)scalar[byte_id] >> shift;\n"
		"  if (shift > 4) { value |= ((uint)scalar[byte_id - 1]) << (8 - shift); }\n"
		"  return value & 0x0f;\n"
		"}\n"
		"static inline uint scalar_window(device const uchar *scalar, uint w) {\n"
		"  uint bit = w * 4;\n"
		"  uint byte_id = 31 - (bit >> 3);\n"
		"  uint shift = bit & 7;\n"
		"  uint value = (uint)scalar[byte_id] >> shift;\n"
		"  if (shift > 4) { value |= ((uint)scalar[byte_id - 1]) << (8 - shift); }\n"
		"  return value & 0x0f;\n"
		"}\n"
		"static inline void comb_load(thread ulong *x, thread ulong *y, thread ulong *z, device const ulong *table, uint w, uint j) {\n"
		"  uint base = (w * COMB_ENTRIES + j) * 12;\n"
		"  for (uint i = 0; i < 4; i++) { x[i] = table[base + i]; y[i] = table[base + 4 + i]; z[i] = table[base + 8 + i]; }\n"
		"}\n"
		"static inline void store_fe_be(thread uchar *out, uint offset, thread const ulong *a) {\n"
		"  for (uint i = 0; i < 4; i++) { ulong limb = a[3 - i]; for (uint j = 0; j < 8; j++) { out[offset + i * 8 + j] = (uchar)(limb >> (56 - 8 * j)); } }\n"
		"}\n"
		"static inline void store_fe_be(device uchar *out, uint offset, thread const ulong *a) {\n"
		"  for (uint i = 0; i < 4; i++) { ulong limb = a[3 - i]; for (uint j = 0; j < 8; j++) { out[offset + i * 8 + j] = (uchar)(limb >> (56 - 8 * j)); } }\n"
		"}\n"
		"static inline void secp256k1_public_key(device const uchar *private_key, device const ulong *table, thread uchar *public_key) {\n"
		"  ulong x[4]; ulong y[4]; ulong z[4];\n"
		"  comb_load(x, y, z, table, 63, scalar_window(private_key, 63));\n"
		"  for (int w = 62; w >= 0; w--) {\n"
		"    ulong tx[4]; ulong ty[4]; ulong tz[4];\n"
		"    comb_load(tx, ty, tz, table, (uint)w, scalar_window(private_key, (uint)w));\n"
		"    jacobian_add(x, y, z, x, y, z, tx, ty, tz);\n"
		"  }\n"
		"  if (fe_is_zero(z)) { for (uint i = 0; i < 64; i++) { public_key[i] = 0; } return; }\n"
		"  ulong zinv[4]; ulong z2[4]; ulong z3[4];\n"
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
		"kernel void count_matches(device const uchar *private_keys [[buffer(0)]], device const uchar *prefix [[buffer(1)]], device const uchar *suffix [[buffer(2)]], device uint *match_indices [[buffer(3)]], device atomic_uint *match_count [[buffer(4)]], device const ulong *comb_table [[buffer(5)]], constant uint &count [[buffer(6)]], constant uint &prefix_len [[buffer(7)]], constant uint &suffix_len [[buffer(8)]], constant uint &max_matches [[buffer(9)]], uint id [[thread_position_in_grid]]) {\n"
		"  if (id >= count) { return; }\n"
		"  thread uchar public_key[64];\n"
		"  secp256k1_public_key(private_keys + id * 32, comb_table, public_key);\n"
		"  ulong state[25];\n"
		"  keccak256_public_key(public_key, state);\n"
		"  bool ok = true;\n"
		"  for (uint i = 0; i < prefix_len; i++) {\n"
		"    if (address_nibble(state, i) != prefix[i]) { ok = false; }\n"
		"  }\n"
		"  for (uint i = 0; i < suffix_len; i++) {\n"
		"    if (address_nibble(state, 40 - suffix_len + i) != suffix[i]) { ok = false; }\n"
		"  }\n"
		"  if (ok) {\n"
		"    uint slot = atomic_fetch_add_explicit(match_count, 1u, memory_order_relaxed);\n"
		"    if (slot < max_matches) { match_indices[slot] = id; }\n"
		"  }\n"
		"}\n"
		"kernel void dump_pubkeys(device const uchar *private_keys [[buffer(0)]], device const ulong *comb_table [[buffer(1)]], device uchar *pubkeys [[buffer(2)]], constant uint &count [[buffer(3)]], uint id [[thread_position_in_grid]]) {\n"
		"  if (id >= count) { return; }\n"
		"  thread uchar public_key[64];\n"
		"  secp256k1_public_key(private_keys + id * 32, comb_table, public_key);\n"
		"  for (uint i = 0; i < 64; i++) { pubkeys[id * 64 + i] = public_key[i]; }\n"
		"}\n";
}

static void* metal_create_match_context(const uint8_t *comb_table, uint32_t comb_table_len, char **error_msg) {
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

		id<MTLBuffer> table_buffer = nil;
		if (comb_table != NULL && comb_table_len > 0) {
			table_buffer = [device newBufferWithLength:(NSUInteger)comb_table_len options:MTLResourceStorageModeShared];
			if (table_buffer == nil) {
				*error_msg = strdup("failed to allocate metal comb table buffer");
				return NULL;
			}
			memcpy([table_buffer contents], comb_table, comb_table_len);
		}

		BlocoMetalMatchContext *context = [BlocoMetalMatchContext new];
		context.device = device;
		context.pipeline = pipeline;
		context.queue = queue;
		context.comb_table_buffer = table_buffer;
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

// metal_ensure_buffer grows a pooled buffer to at least the requested length.
// ARC releases the previous buffer when the strong property is reassigned.
static id<MTLBuffer> metal_ensure_buffer(id<MTLDevice> device, id<MTLBuffer> buffer, NSUInteger length, uint32_t *capacity, uint32_t requested) {
	if (buffer != nil && *capacity >= requested) {
		return buffer;
	}
	*capacity = requested;
	return [device newBufferWithLength:length options:MTLResourceStorageModeShared];
}

static int metal_run_match(void *context_ptr, const uint8_t *private_keys, uint32_t count, const uint8_t *prefix, uint32_t prefix_len, const uint8_t *suffix, uint32_t suffix_len, uint32_t *matches, uint32_t *match_indices, uint32_t max_matches, double *buffer_ns, double *kernel_ns, char **error_msg) {
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
		NSUInteger prefix_byte_count = prefix_len == 0 ? 1 : prefix_len;
		NSUInteger suffix_byte_count = suffix_len == 0 ? 1 : suffix_len;
		NSUInteger match_index_byte_count = (NSUInteger)max_matches * sizeof(uint32_t);

		// Reuse pooled buffers, growing them only when a larger batch arrives.
		uint32_t private_key_capacity = context.private_key_capacity;
		uint32_t prefix_capacity = context.prefix_capacity;
		uint32_t suffix_capacity = context.suffix_capacity;
		uint32_t match_index_capacity = context.match_index_capacity;
		context.private_key_buffer = metal_ensure_buffer(device, context.private_key_buffer, private_key_byte_count, &private_key_capacity, count);
		context.prefix_buffer = metal_ensure_buffer(device, context.prefix_buffer, prefix_byte_count, &prefix_capacity, prefix_len == 0 ? 1 : prefix_len);
		context.suffix_buffer = metal_ensure_buffer(device, context.suffix_buffer, suffix_byte_count, &suffix_capacity, suffix_len == 0 ? 1 : suffix_len);
		context.match_index_buffer = metal_ensure_buffer(device, context.match_index_buffer, match_index_byte_count, &match_index_capacity, max_matches);
		context.private_key_capacity = private_key_capacity;
		context.prefix_capacity = prefix_capacity;
		context.suffix_capacity = suffix_capacity;
		context.match_index_capacity = match_index_capacity;
		if (context.match_count_buffer == nil) {
			context.match_count_buffer = [device newBufferWithLength:sizeof(uint32_t) options:MTLResourceStorageModeShared];
		}
		id<MTLBuffer> private_key_buffer = context.private_key_buffer;
		id<MTLBuffer> prefix_buffer = context.prefix_buffer;
		id<MTLBuffer> suffix_buffer = context.suffix_buffer;
		id<MTLBuffer> match_index_buffer = context.match_index_buffer;
		id<MTLBuffer> match_count_buffer = context.match_count_buffer;
		if (private_key_buffer == nil || prefix_buffer == nil || suffix_buffer == nil || match_index_buffer == nil || match_count_buffer == nil) {
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
		uint32_t *match_count_contents = (uint32_t *)[match_count_buffer contents];
		*match_count_contents = 0;
		memset([match_index_buffer contents], 0, match_index_byte_count);
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
		[encoder setBuffer:match_index_buffer offset:0 atIndex:3];
		[encoder setBuffer:match_count_buffer offset:0 atIndex:4];
		[encoder setBuffer:context.comb_table_buffer offset:0 atIndex:5];
		[encoder setBytes:&count length:sizeof(count) atIndex:6];
		[encoder setBytes:&prefix_len length:sizeof(prefix_len) atIndex:7];
		[encoder setBytes:&suffix_len length:sizeof(suffix_len) atIndex:8];
		[encoder setBytes:&max_matches length:sizeof(max_matches) atIndex:9];

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

		*matches = *match_count_contents;
		memcpy(match_indices, [match_index_buffer contents], match_index_byte_count);
		metal_zero_buffer(private_key_buffer);
		metal_zero_buffer(prefix_buffer);
		metal_zero_buffer(suffix_buffer);
		*buffer_ns = (buffer_end - buffer_start) * 1000000000.0;
		*kernel_ns = (end - start) * 1000000000.0;
		return 0;
	}
}

// metal_dump_pubkeys is a diagnostic helper used by tests to read the raw
// public keys the kernel derives for a batch of private keys.
static int metal_dump_pubkeys(const char *kernel_name, const uint8_t *comb_table, uint32_t comb_table_len, const uint8_t *private_keys, uint32_t count, uint32_t out_stride, uint8_t *out, char **error_msg) {
	@autoreleasepool {
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) {
			*error_msg = strdup("metal device not available");
			return 1;
		}
		NSError *error = nil;
		id<MTLLibrary> library = [device newLibraryWithSource:metal_match_source() options:nil error:&error];
		if (library == nil) {
			*error_msg = metal_copy_error(error);
			return 2;
		}
		NSString *name = [NSString stringWithUTF8String:kernel_name];
		id<MTLFunction> function = [library newFunctionWithName:name];
		if (function == nil) {
			*error_msg = strdup("metal dump function not found");
			return 3;
		}
		id<MTLComputePipelineState> pipeline = [device newComputePipelineStateWithFunction:function error:&error];
		if (pipeline == nil) {
			*error_msg = metal_copy_error(error);
			return 4;
		}
		id<MTLCommandQueue> queue = [device newCommandQueue];
		if (queue == nil) {
			*error_msg = strdup("failed to create metal command queue");
			return 5;
		}

		id<MTLBuffer> key_buffer = [device newBufferWithLength:(NSUInteger)count * 32 options:MTLResourceStorageModeShared];
		id<MTLBuffer> table_buffer = [device newBufferWithLength:(NSUInteger)comb_table_len options:MTLResourceStorageModeShared];
		id<MTLBuffer> out_buffer = [device newBufferWithLength:(NSUInteger)count * out_stride options:MTLResourceStorageModeShared];
		if (key_buffer == nil || table_buffer == nil || out_buffer == nil) {
			*error_msg = strdup("failed to allocate dump buffers");
			return 6;
		}
		memcpy([key_buffer contents], private_keys, (NSUInteger)count * 32);
		memcpy([table_buffer contents], comb_table, comb_table_len);

		id<MTLCommandBuffer> cb = [queue commandBuffer];
		id<MTLComputeCommandEncoder> enc = [cb computeCommandEncoder];
		if (cb == nil || enc == nil) {
			*error_msg = strdup("failed to create metal command encoder");
			return 7;
		}
		[enc setComputePipelineState:pipeline];
		[enc setBuffer:key_buffer offset:0 atIndex:0];
		[enc setBuffer:table_buffer offset:0 atIndex:1];
		[enc setBuffer:out_buffer offset:0 atIndex:2];
		[enc setBytes:&count length:sizeof(count) atIndex:3];
		NSUInteger width = (NSUInteger)count;
		NSUInteger tg = pipeline.maxTotalThreadsPerThreadgroup;
		if (tg > width) { tg = width; }
		[enc dispatchThreads:MTLSizeMake(width, 1, 1) threadsPerThreadgroup:MTLSizeMake(tg, 1, 1)];
		[enc endEncoding];
		[cb commit];
		[cb waitUntilCompleted];
		if ([cb status] == MTLCommandBufferStatusError) {
			*error_msg = metal_copy_error([cb error]);
			return 8;
		}
		memcpy(out, [out_buffer contents], (NSUInteger)count * out_stride);
		return 0;
	}
}

*/
import "C"

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"
	"unsafe"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/crypto/sha3"

	"bloco-vgen/pkg/wallet"
)

type MetalEngine struct {
	deviceName string
	context    unsafe.Pointer
}

type metalGenerationCandidate struct {
	Found      bool
	PrivateKey [32]byte
	PublicKey  [65]byte
	Address    string
	Stages     stageTotals
	RawMatches uint32
}

func MetalAvailable() bool {
	return C.metal_available() == 1
}

func MetalUnavailableReason() string {
	return "metal device not available; using cpu"
}

func MetalDeviceName() string {
	if !MetalAvailable() {
		return ""
	}
	context, err := newMetalMatchContext()
	if err != nil {
		return ""
	}
	defer C.metal_release_match_context(context)

	name, err := metalContextDeviceName(context)
	if err != nil {
		return ""
	}
	return name
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
	validationMode, err := NormalizeMetalValidationMode(options.MetalValidation)
	if err != nil {
		return nil, err
	}
	options.MetalValidation = validationMode

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
		batchAttempts, batchMatches, batchStages, batchMetalBufferDuration, batchKernelDuration, err := runMetalHybridBatch(e.context, benchmarkCtx, currentBatchSize, prefixNibbles, suffixNibbles, validationMode)
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

func (e *MetalEngine) GenerateWallet(ctx context.Context, options GenerationOptions, sampleInterval time.Duration, onSample func(GenerationSample)) (*wallet.GenerationResult, error) {
	if err := ValidateMetalGenerationCriteria(options.Criteria); err != nil {
		return nil, err
	}
	validationMode, err := NormalizeMetalValidationMode(options.MetalValidation)
	if err != nil {
		return nil, err
	}
	options.MetalValidation = validationMode

	prefixNibbles, err := patternNibbles(options.Criteria.Prefix)
	if err != nil {
		return nil, err
	}
	suffixNibbles, err := patternNibbles(options.Criteria.Suffix)
	if err != nil {
		return nil, err
	}

	batchSize := options.BatchSize
	if batchSize <= 0 {
		batchSize = DefaultMetalBatchSize
	}

	start := time.Now()
	lastSample := start
	lastSampleAttempts := int64(0)
	totalAttempts := int64(0)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		privateKeys, _, err := generateEthereumPrivateKeyBatch(ctx, batchSize)
		if err != nil {
			return nil, err
		}
		batchAttempts := int64(len(privateKeys) / 32)

		gpuMatches, matchIndices, _, _, err := runMetalMatch(e.context, privateKeys, prefixNibbles, suffixNibbles)
		if err != nil {
			zeroBytes(privateKeys)
			return nil, err
		}

		candidate, err := findValidatedMetalGenerationCandidate(privateKeys, prefixNibbles, suffixNibbles, gpuMatches, matchIndices, options.Criteria, validationMode)
		zeroBytes(privateKeys)
		if err != nil {
			zeroBytes(candidate.PrivateKey[:])
			return nil, err
		}

		totalAttempts += batchAttempts
		now := time.Now()
		if onSample != nil && (sampleInterval <= 0 || now.Sub(lastSample) >= sampleInterval) {
			elapsed := now.Sub(lastSample)
			speed := 0.0
			if elapsed > 0 {
				speed = float64(totalAttempts-lastSampleAttempts) / elapsed.Seconds()
			}
			onSample(GenerationSample{Attempts: totalAttempts, Speed: speed, Elapsed: now.Sub(start)})
			lastSample = now
			lastSampleAttempts = totalAttempts
		}

		if !candidate.Found {
			continue
		}

		privateKeyHex := hex.EncodeToString(candidate.PrivateKey[:])
		publicKeyHex := hex.EncodeToString(candidate.PublicKey[:])
		zeroBytes(candidate.PrivateKey[:])

		result := &wallet.GenerationResult{
			Wallet: &wallet.Wallet{
				Address:    candidate.Address,
				PublicKey:  publicKeyHex,
				PrivateKey: privateKeyHex,
				Network:    options.Criteria.Network,
				CreatedAt:  time.Now(),
			},
			Attempts:        totalAttempts,
			Duration:        time.Since(start),
			Engine:          NameMetal,
			RequestedEngine: options.RequestedEngine,
			FallbackReason:  options.FallbackReason,
			DeviceName:      e.deviceName,
			BatchSize:       batchSize,
			MetalValidation: validationMode,
		}
		if !MatchesCriteria(result.Wallet.Address, options.Criteria) {
			return nil, fmt.Errorf("metal generation validation failed for final wallet")
		}
		if onSample != nil {
			speed := 0.0
			if result.Duration > 0 {
				speed = float64(result.Attempts) / result.Duration.Seconds()
			}
			onSample(GenerationSample{Attempts: result.Attempts, Speed: speed, Elapsed: result.Duration})
		}
		return result, nil
	}
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
		MetalValidation:        options.MetalValidation,
		Matches:                totalMatches,
		TotalAttempts:          totalAttempts,
		TotalDuration:          totalDuration,
		AverageSpeed:           averageSpeed,
		CPUThroughput:          throughputForDuration(totalAttempts, stages.Scalar+stages.Hash+stages.Match),
		GPUThroughput:          throughputForDuration(totalAttempts, kernelDuration),
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
	table, err := buildMetalCombTable()
	if err != nil {
		return nil, err
	}
	context := C.metal_create_match_context((*C.uint8_t)(unsafe.Pointer(&table[0])), C.uint32_t(len(table)), &errorMsg)
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

// metalCombTableSize is 64 windows x 16 entries x (X, Y, Z) x 4 limbs of
// uint64 in little-endian byte order.
const metalCombTableSize = 64 * 16 * 3 * 4 * 8

// buildMetalCombTable precomputes the fixed-base comb table used by the Metal
// kernel: entry [w][j] is the Jacobian point (j * 2^(4w)) * G with Z = 1
// (except j == 0, which is the point at infinity with Z = 0).
func buildMetalCombTable() ([]byte, error) {
	table := make([]byte, metalCombTableSize)
	order := secp256k1.S256().Params().N
	for w := 0; w < 64; w++ {
		for j := 0; j < 16; j++ {
			s := new(big.Int).Lsh(big.NewInt(int64(j)), uint(4*w))
			s.Mod(s, order)
			base := (w*16 + j) * 12 // ulong units (3 coords x 4 limbs)
			if s.Sign() == 0 {
				// point at infinity: X = Y = Z = 0
				continue
			}
			var kBytes [32]byte
			s.FillBytes(kBytes[:])
			var kScalar secp256k1.ModNScalar
			kScalar.SetByteSlice(kBytes[:])
			var point secp256k1.JacobianPoint
			secp256k1.ScalarBaseMultNonConst(&kScalar, &point)
			point.ToAffine()
			point.Z.SetInt(1)
			writeCombField(table, base*8, &point.X)
			writeCombField(table, (base+4)*8, &point.Y)
			writeCombField(table, (base+8)*8, &point.Z)
		}
	}
	return table, nil
}

// writeCombField serializes a field value as 4 little-endian 64-bit limbs,
// matching the kernel's fe representation: limb[i] holds bits [64i, 64i+64).
func writeCombField(out []byte, offset int, value *secp256k1.FieldVal) {
	bytes := value.Bytes() // 32-byte big-endian
	for i := 0; i < 4; i++ {
		limb := binary.BigEndian.Uint64(bytes[24-8*i : 32-8*i])
		binary.LittleEndian.PutUint64(out[offset+i*8:], limb)
	}
}

// dumpMetalPubkeys is a diagnostic helper used by tests to read the raw
// public keys the kernel derives for a batch of private keys.
func dumpMetalPubkeys(table []byte, privateKeys []byte) ([]byte, error) {
	return dumpMetalKernel("dump_pubkeys", table, privateKeys, 64)
}

func dumpMetalKernel(kernelName string, table []byte, privateKeys []byte, stride int) ([]byte, error) {
	if len(privateKeys)%32 != 0 {
		return nil, fmt.Errorf("private keys must be a multiple of 32 bytes")
	}
	count := len(privateKeys) / 32
	out := make([]byte, count*stride)
	var errorMsg *C.char
	kernelC := C.CString(kernelName)
	defer C.free(unsafe.Pointer(kernelC))
	code := C.metal_dump_pubkeys(kernelC, (*C.uint8_t)(unsafe.Pointer(&table[0])), C.uint32_t(len(table)), (*C.uint8_t)(unsafe.Pointer(&privateKeys[0])), C.uint32_t(count), C.uint32_t(stride), (*C.uint8_t)(unsafe.Pointer(&out[0])), &errorMsg)
	if errorMsg != nil {
		defer C.free(unsafe.Pointer(errorMsg))
	}
	if code != 0 {
		if errorMsg != nil {
			return nil, fmt.Errorf("dump kernel failed: %s", C.GoString(errorMsg))
		}
		return nil, fmt.Errorf("dump kernel failed with code %d", int(code))
	}
	return out, nil
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
			zeroBytes(privateKeys)
			return nil, totals, ctx.Err()
		default:
		}

		privateKey, entropyDuration, err := generateEthereumPrivateKeyAttempt()
		if err != nil {
			zeroBytes(privateKeys)
			return nil, totals, err
		}
		copy(privateKeys[i*32:(i+1)*32], privateKey[:])
		zeroBytes(privateKey[:])
		totals.Entropy += entropyDuration
	}

	return privateKeys, totals, nil
}

func runMetalHybridBatch(context unsafe.Pointer, ctx context.Context, attempts int, prefix []byte, suffix []byte, validationMode string) (int, uint32, stageTotals, time.Duration, time.Duration, error) {
	privateKeys, stages, err := generateEthereumPrivateKeyBatch(ctx, attempts)
	if err != nil {
		return 0, 0, stages, 0, 0, err
	}
	defer zeroBytes(privateKeys)

	gpuMatches, matchIndices, metalBufferDuration, kernelDuration, err := runMetalMatch(context, privateKeys, prefix, suffix)
	if err != nil {
		return 0, 0, stages, metalBufferDuration, kernelDuration, err
	}
	validationStages, err := validateMetalBatch(privateKeys, prefix, suffix, gpuMatches, matchIndices, validationMode)
	if err != nil {
		return 0, 0, stages, metalBufferDuration, kernelDuration, err
	}
	stages.Scalar += validationStages.Scalar
	stages.Hash += validationStages.Hash
	stages.Match += validationStages.Match

	return len(privateKeys) / 32, gpuMatches, stages, metalBufferDuration, kernelDuration, nil
}

func validateMetalBatch(privateKeys []byte, prefix []byte, suffix []byte, gpuMatches uint32, matchIndices []uint32, validationMode string) (stageTotals, error) {
	validationMode, err := NormalizeMetalValidationMode(validationMode)
	if err != nil {
		return stageTotals{}, err
	}
	if validationMode != MetalValidationFull {
		return stageTotals{}, fmt.Errorf("unsupported metal validation mode %q", validationMode)
	}

	// The kernel reports the index of every candidate that matched; only
	// those (typically a handful per batch) are re-verified on the CPU
	// instead of re-deriving every key in the batch.
	stages := stageTotals{}
	hasher := sha3.NewLegacyKeccak256()
	for i := uint32(0); i < gpuMatches && i < uint32(len(matchIndices)); i++ {
		index := matchIndices[i]
		privateKey := privateKeys[index*32 : (index+1)*32]

		stageStart := time.Now()
		x, y := ethcrypto.S256().ScalarBaseMult(privateKey)
		stages.Scalar += time.Since(stageStart)

		var publicKey [64]byte
		x.FillBytes(publicKey[:32])
		y.FillBytes(publicKey[32:])

		stageStart = time.Now()
		address := EthereumAddressBytesFromPublicKey(publicKey[:], hasher)
		stages.Hash += time.Since(stageStart)

		stageStart = time.Now()
		if !addressMatches(address[:], prefix, suffix) {
			return stages, fmt.Errorf("metal match validation failed: gpu index %d is not a match", index)
		}
		stages.Match += time.Since(stageStart)
	}
	return stages, nil
}

func findValidatedMetalGenerationCandidate(privateKeys []byte, prefix []byte, suffix []byte, gpuMatches uint32, matchIndices []uint32, criteria wallet.GenerationCriteria, validationMode string) (metalGenerationCandidate, error) {
	validationMode, err := NormalizeMetalValidationMode(validationMode)
	if err != nil {
		return metalGenerationCandidate{}, err
	}
	if validationMode != MetalValidationFull {
		return metalGenerationCandidate{}, fmt.Errorf("unsupported metal validation mode %q", validationMode)
	}

	candidate := metalGenerationCandidate{}
	candidate.RawMatches = gpuMatches
	hasher := sha3.NewLegacyKeccak256()
	for i := uint32(0); i < gpuMatches && i < uint32(len(matchIndices)); i++ {
		index := matchIndices[i]
		privateKey := privateKeys[index*32 : (index+1)*32]

		stageStart := time.Now()
		x, y := ethcrypto.S256().ScalarBaseMult(privateKey)
		candidate.Stages.Scalar += time.Since(stageStart)

		var publicKey [64]byte
		x.FillBytes(publicKey[:32])
		y.FillBytes(publicKey[32:])

		stageStart = time.Now()
		address := EthereumAddressBytesFromPublicKey(publicKey[:], hasher)
		candidate.Stages.Hash += time.Since(stageStart)

		stageStart = time.Now()
		rawMatched := addressMatches(address[:], prefix, suffix)
		if !rawMatched {
			zeroBytes(candidate.PrivateKey[:])
			return metalGenerationCandidate{}, fmt.Errorf("metal match validation failed: gpu index %d is not a match", index)
		}
		fullMatched := MatchesCriteria(FormatEthereumAddressBytes(address), criteria)
		candidate.Stages.Match += time.Since(stageStart)

		if fullMatched && !candidate.Found {
			candidate.Found = true
			copy(candidate.PrivateKey[:], privateKey)
			candidate.PublicKey[0] = 4
			copy(candidate.PublicKey[1:33], publicKey[:32])
			copy(candidate.PublicKey[33:], publicKey[32:])
			candidate.Address = FormatEthereumAddressBytes(address)
			if criteria.IsChecksum {
				candidate.Address = ChecksumAddress(candidate.Address)
			}
		}
	}
	return candidate, nil
}

func runMetalMatch(context unsafe.Pointer, privateKeys []byte, prefix []byte, suffix []byte) (uint32, []uint32, time.Duration, time.Duration, error) {
	count, err := metalPrivateKeyBatchCount(privateKeys, maxMetalBatchCandidates())
	if err != nil {
		return 0, nil, 0, 0, err
	}
	if err := validatePrivateKeyBatch(privateKeys); err != nil {
		return 0, nil, 0, 0, err
	}

	var bufferNS C.double
	var kernelNS C.double
	var errorMsg *C.char
	var matches C.uint32_t
	matchIndices := make([]uint32, count)
	privateKeyPtr := (*C.uint8_t)(unsafe.Pointer(&privateKeys[0]))
	prefixPtr := cBytePointer(prefix)
	suffixPtr := cBytePointer(suffix)
	code := C.metal_run_match(context, privateKeyPtr, C.uint32_t(count), prefixPtr, C.uint32_t(len(prefix)), suffixPtr, C.uint32_t(len(suffix)), &matches, (*C.uint32_t)(unsafe.Pointer(&matchIndices[0])), C.uint32_t(count), &bufferNS, &kernelNS, &errorMsg)
	if errorMsg != nil {
		defer C.free(unsafe.Pointer(errorMsg))
	}
	if code != 0 {
		if errorMsg != nil {
			return 0, nil, 0, 0, fmt.Errorf("metal match kernel failed: %s", C.GoString(errorMsg))
		}
		return 0, nil, 0, 0, fmt.Errorf("metal match kernel failed with code %d", int(code))
	}

	return uint32(matches), matchIndices, durationFromNanoseconds(float64(bufferNS)), durationFromNanoseconds(float64(kernelNS)), nil
}

func validatePrivateKeyBatch(privateKeys []byte) error {
	for i := 0; i < len(privateKeys); i += 32 {
		if !validSecp256k1PrivateKey(privateKeys[i : i+32]) {
			return fmt.Errorf("metal match private key %d is outside secp256k1 range", i/32)
		}
	}
	return nil
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
