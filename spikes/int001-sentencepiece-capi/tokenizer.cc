//go:build linux && cgo && sentencepiece

#include "tokenizer.h"

#include <new>
#include <string>
#include <vector>

#include "sentencepiece_processor.h"

struct fp_sentencepiece {
  sentencepiece::SentencePieceProcessor processor;
};

fp_sentencepiece* fp_sentencepiece_open(const char* path) {
  if (path == nullptr) return nullptr;
  fp_sentencepiece* value = new (std::nothrow) fp_sentencepiece();
  if (value == nullptr) return nullptr;
  if (!value->processor.Load(path).ok()) {
    delete value;
    return nullptr;
  }
  return value;
}

void fp_sentencepiece_close(fp_sentencepiece* value) { delete value; }

int fp_sentencepiece_piece_size(const fp_sentencepiece* value) {
  return value == nullptr ? -1 : value->processor.GetPieceSize();
}

int fp_sentencepiece_unk_id(const fp_sentencepiece* value) {
  return value == nullptr ? -1 : value->processor.unk_id();
}

int fp_sentencepiece_eos_id(const fp_sentencepiece* value) {
  return value == nullptr ? -1 : value->processor.eos_id();
}

int fp_sentencepiece_encode(const fp_sentencepiece* value, const char* input,
                            size_t input_size, int32_t* output, size_t output_capacity) {
  if (value == nullptr || input == nullptr || output == nullptr) return -1;
  std::vector<int> ids;
  if (!value->processor.Encode(std::string(input, input_size), &ids).ok()) return -2;
  if (ids.size() > output_capacity) return -3;
  for (size_t index = 0; index < ids.size(); ++index) output[index] = ids[index];
  return static_cast<int>(ids.size());
}
