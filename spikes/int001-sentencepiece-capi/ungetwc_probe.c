#include <stdio.h>
#include <wchar.h>

int main(void) {
  FILE *stream = tmpfile();
  if (stream == NULL) {
    return 2;
  }
  wint_t result = ungetwc(L'x', stream);
  fclose(stream);
  return result == WEOF ? 3 : 0;
}
