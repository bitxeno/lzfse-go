/*
 * C shim for the lzfse-go shared library.
 *
 * lzfse upstream declares lzvn_decode_buffer in lzfse_internal.h but never
 * defines it, so we provide the one-shot wrapper around the LZVN low-level
 * decoder. The libzbitmap functions return negative error codes and write
 * through a size_t out-parameter; wrap them into plain size_t returns so
 * every symbol in the library follows the same convention: take pointers and
 * sizes in, return the number of output bytes, or 0 on failure.
 */
#include <stdlib.h>
#include <string.h>

#include "lzfse/lzfse.h"
#include "lzfse/lzfse_internal.h"
#include "lzfse/lzvn_decode_base.h"
#include "libzbitmap/libzbitmap.h"

size_t lzvn_decode_buffer(void *__restrict dst, size_t dst_size,
                          const void *__restrict src, size_t src_size) {
  // Init LZVN decoder state
  lzvn_decoder_state dstate;
  memset(&dstate, 0x00, sizeof(dstate));
  dstate.src = src;
  dstate.src_end = (const unsigned char *)src + src_size;

  dstate.dst_begin = dst;
  dstate.dst = dst;
  dstate.dst_end = (unsigned char *)dst + dst_size;

  dstate.d_prev = 0;
  dstate.end_of_stream = 0;

  // Run LZVN decoder
  lzvn_decode(&dstate);

  // This is how much we decompressed
  return dstate.dst - (unsigned char *)dst;
}

size_t lzfse_zbm_compress(void *dst, size_t dst_size, const void *src,
                          size_t src_size) {
  size_t out_len = 0;
  if (zbm_compress(dst, dst_size, src, src_size, &out_len) != 0) {
    return 0;
  }
  return out_len;
}

size_t lzfse_zbm_decompress(void *dst, size_t dst_size, const void *src,
                            size_t src_size) {
  size_t out_len = 0;
  if (zbm_decompress(dst, dst_size, src, src_size, &out_len) != 0) {
    return 0;
  }
  return out_len;
}
