#pragma once

#ifdef __cplusplus
#include "M5GFX.h"

m5gfx::M5GFX &display_get(void);
bool display_init_m5gfx(void);
void display_present_canvas(M5Canvas &canvas);
#endif
