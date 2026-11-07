#ifndef librouter_h
#define librouter_h

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

extern int RouterStart(char* cfgJSON);
extern void RouterStop(void);
extern void RouterWritePacket(char* data, int n);
extern int RouterReadPacket(char* buf, int maxLen);

#ifdef __cplusplus
}
#endif

#endif /* librouter_h */

