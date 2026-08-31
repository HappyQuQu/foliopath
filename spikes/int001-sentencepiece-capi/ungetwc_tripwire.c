#include <stdio.h>
#include <unistd.h>
#include <wchar.h>

wint_t ungetwc(wint_t character, FILE *stream) {
  (void)character;
  (void)stream;
  static const char message[] = "foliopath INT-001 ungetwc tripwire triggered\n";
  (void)write(STDERR_FILENO, message, sizeof(message) - 1);
  _exit(86);
}
