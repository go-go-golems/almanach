#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

void backlight_prepare_for_init(void);
void backlight_enable_after_init(void);
void backlight_set_brightness(uint8_t brightness);

#ifdef __cplusplus
}
#endif
