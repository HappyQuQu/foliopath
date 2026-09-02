#ifndef FOLIOPATH_SENTENCEPIECE_TOKENIZER_H
#define FOLIOPATH_SENTENCEPIECE_TOKENIZER_H

#include <stddef.h>
#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif
typedef struct fp_sentencepiece fp_sentencepiece;
fp_sentencepiece* fp_sentencepiece_open(const char* path);
void fp_sentencepiece_close(fp_sentencepiece* value);
int fp_sentencepiece_piece_size(const fp_sentencepiece* value);
int fp_sentencepiece_unk_id(const fp_sentencepiece* value);
int fp_sentencepiece_eos_id(const fp_sentencepiece* value);
int fp_sentencepiece_encode(const fp_sentencepiece* value, const char* input,
                            size_t input_size, int32_t* output, size_t output_capacity);

#ifdef __cplusplus
}
#endif

#endif
