/*
 * Circuit garbling using AES-NI instructions.
 */

#include <wmmintrin.h>
#include <stdio.h>
#include <stdint.h>
#include <string.h>
#include <sys/time.h>

#define AES_BLOCK_SIZE 16

struct cipher
{
  __m128i key[15];
};

typedef struct cipher cipher;

void
make_cipher(cipher *cipher, const unsigned char key[AES_BLOCK_SIZE])
{
  cipher->key[0] = _mm_loadu_si128((__m128i *)key);
  cipher->key[1] = _mm_aeskeygenassist_si128(cipher->key[0], 0x01);
  cipher->key[2] = _mm_aeskeygenassist_si128(cipher->key[1], 0x02);
  cipher->key[3] = _mm_aeskeygenassist_si128(cipher->key[2], 0x04);
  cipher->key[4] = _mm_aeskeygenassist_si128(cipher->key[3], 0x08);
  cipher->key[5] = _mm_aeskeygenassist_si128(cipher->key[4], 0x10);
  cipher->key[6] = _mm_aeskeygenassist_si128(cipher->key[5], 0x20);
  cipher->key[7] = _mm_aeskeygenassist_si128(cipher->key[6], 0x40);
  cipher->key[8] = _mm_aeskeygenassist_si128(cipher->key[7], 0x80);
  cipher->key[9] = _mm_aeskeygenassist_si128(cipher->key[8], 0x1B);
  cipher->key[10] = _mm_aeskeygenassist_si128(cipher->key[9], 0x36);
  cipher->key[11] = _mm_aeskeygenassist_si128(cipher->key[10], 0x6C);
  cipher->key[12] = _mm_aeskeygenassist_si128(cipher->key[11], 0xD8);
  cipher->key[13] = _mm_aeskeygenassist_si128(cipher->key[12], 0xAB);
  cipher->key[14] = _mm_aeskeygenassist_si128(cipher->key[13], 0x4D);
}

__m128i
makeK(__m128i a, __m128i b, uint32_t t)
{
  __m128i k;
  __m128i tweak;

  a = _mm_slli_epi32(a, 1);
  b = _mm_slli_epi32(b, 2);

  k = _mm_xor_si128(a, b);

  tweak = _mm_set1_epi32(t);

  return _mm_xor_si128(k, tweak);
}

static inline __m128i
garble(cipher *cipher, __m128i a, __m128i b, __m128i c, uint32_t t)
{
  __m128i k;
  __m128i pi;

  k = _mm_xor_si128(_mm_xor_si128(_mm_slli_epi32(a, 1),
                                  _mm_slli_epi32(b, 2)),
                    _mm_set1_epi32(t));

  /* Perform the AES encryption rounds. */
  pi = _mm_xor_si128(k, cipher->key[0]);
  pi = _mm_aesenc_si128(pi, cipher->key[1]);
  pi = _mm_aesenc_si128(pi, cipher->key[2]);
  pi = _mm_aesenc_si128(pi, cipher->key[3]);
  pi = _mm_aesenc_si128(pi, cipher->key[4]);
  pi = _mm_aesenc_si128(pi, cipher->key[5]);
  pi = _mm_aesenc_si128(pi, cipher->key[6]);
  pi = _mm_aesenc_si128(pi, cipher->key[7]);
  pi = _mm_aesenc_si128(pi, cipher->key[8]);
  pi = _mm_aesenc_si128(pi, cipher->key[9]);
  pi = _mm_aesenc_si128(pi, cipher->key[10]);
  pi = _mm_aesenc_si128(pi, cipher->key[11]);
  pi = _mm_aesenc_si128(pi, cipher->key[12]);
  pi = _mm_aesenc_si128(pi, cipher->key[13]);
  pi = _mm_aesenclast_si128(pi, cipher->key[14]);

  return _mm_xor_si128(_mm_xor_si128(pi, k), c);
}

int
main()
{
  const unsigned char key[AES_BLOCK_SIZE] = "0123456789ABCDEF";
  cipher cipher = {0};
  struct timeval begin, end;
  int64_t d;
  int rounds = 23802664;
  __m128i a = _mm_set1_epi32(42);
  __m128i b = _mm_set1_epi32(43);
  __m128i c = _mm_set1_epi32(44);

  make_cipher(&cipher, key);

  gettimeofday(&begin, NULL);
  for (int i = 0; i < rounds; i++)
    garble(&cipher, a, b, c, (uint32_t) i);
  gettimeofday(&end, NULL);

  d = (int64_t) end.tv_sec - (int64_t) begin.tv_sec;
  d *= 1000000;
  d += (int64_t) end.tv_usec - (int64_t) begin.tv_usec;
  d *= 1000;
  d *= 100;

  d /= rounds;

  printf("%-20s\t%d\t\t%.2f ns/op\n", "AES-NI", rounds, (double) d / 100);

  return 0;
}
