#ifndef QQCLIPUT_H
#define QQCLIPUT_H

#include <stdint.h>

uint32_t find_qq_window(void);
char* ocr_window(uint32_t window_id);
char* ocr_window_json(uint32_t window_id);
int window_exists(uint32_t window_id);
int is_qq_frontmost(void);
char* get_window_bounds(uint32_t window_id);
char* ocr_window_region_json(uint32_t window_id, double nx, double ny, double nw, double nh);

#endif
