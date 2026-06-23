#ifndef QQCLIPUT_H
#define QQCLIPUT_H

#include <stdint.h>

uint32_t find_qq_window(void);
char* ocr_window(uint32_t window_id);
int window_exists(uint32_t window_id);
int is_qq_frontmost(void);

#endif
