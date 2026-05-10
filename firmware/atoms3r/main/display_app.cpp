/*
 * Text-first AtomS3R display UI for Almanach.
 */

#include "display_app.h"

#include <inttypes.h>
#include <stdio.h>
#include <string.h>

#include "sdkconfig.h"

#include "esp_heap_caps.h"
#include "esp_log.h"

#include "M5GFX.h"

#include "backlight.h"
#include "display_hal.h"

static const char *TAG = "display_app";

static bool s_inited = false;
static const int s_w = CONFIG_ALMANACH_ATOMS3R_LCD_HRES;
static const int s_h = CONFIG_ALMANACH_ATOMS3R_LCD_VRES;
static M5Canvas s_canvas(&display_get());

static void set_text(uint16_t fg)
{
    s_canvas.setTextColor(fg, TFT_BLACK);
    s_canvas.setTextSize(1);
}

static void print_clipped(const char *s)
{
    if (!s || !s[0]) {
        s_canvas.println("-");
        return;
    }
    char tmp[22];
    snprintf(tmp, sizeof(tmp), "%s", s);
    s_canvas.println(tmp);
}

esp_err_t display_app_init(void)
{
#if !CONFIG_ALMANACH_ATOMS3R_DISPLAY_ENABLE
    return ESP_ERR_NOT_SUPPORTED;
#else
    if (s_inited) {
        return ESP_OK;
    }

    ESP_LOGI(TAG, "display boot; free_heap=%" PRIu32 " dma_free=%" PRIu32,
             esp_get_free_heap_size(),
             (uint32_t)heap_caps_get_free_size(MALLOC_CAP_DMA));

    backlight_prepare_for_init();
    if (!display_init_m5gfx()) {
        ESP_LOGE(TAG, "display init failed");
        return ESP_FAIL;
    }
    backlight_enable_after_init();

#if CONFIG_ALMANACH_ATOMS3R_CANVAS_USE_PSRAM
    s_canvas.setPsram(true);
#else
    s_canvas.setPsram(false);
#endif
    s_canvas.setColorDepth(16);
    void *buf = s_canvas.createSprite(s_w, s_h);
    if (!buf) {
        ESP_LOGE(TAG, "canvas createSprite failed (%dx%d)", s_w, s_h);
        return ESP_ERR_NO_MEM;
    }

    ESP_LOGI(TAG, "canvas ok: %u bytes", (unsigned)s_canvas.bufferLength());
    s_inited = true;
    return ESP_OK;
#endif
}

bool display_app_is_ready(void)
{
    return s_inited;
}

void display_app_show_boot(const char *line1, const char *line2)
{
    if (!s_inited) {
        return;
    }
    s_canvas.fillScreen(TFT_BLACK);
    s_canvas.setCursor(0, 0);
    set_text(TFT_CYAN);
    s_canvas.println("ALMANACH");
    set_text(TFT_WHITE);
    if (line1) {
        s_canvas.println(line1);
    }
    if (line2) {
        s_canvas.println(line2);
    }
    display_present_canvas(s_canvas);
}

void display_app_show_status(const display_status_t *st)
{
    if (!s_inited || !st) {
        return;
    }

    s_canvas.fillScreen(TFT_BLACK);
    s_canvas.setCursor(0, 0);
    set_text(TFT_CYAN);
    s_canvas.println("ALMANACH");

    if (st->wifi_connected) {
        set_text(TFT_GREEN);
        s_canvas.println("ONLINE");
        set_text(TFT_WHITE);
        print_clipped(st->ip);
        s_canvas.println("Web ready");
    } else if (st->provisioning_running) {
        set_text(TFT_YELLOW);
        s_canvas.println("PAIR BLE");
        set_text(TFT_WHITE);
        print_clipped(st->service_name);
        s_canvas.print("PoP ");
        print_clipped(st->pop);
        if (st->provisioning_client_connected) {
            s_canvas.println("Client: yes");
        }
        if (st->provisioning_security_ok) {
            s_canvas.println("Security OK");
        }
    } else if (st->provisioned) {
        set_text(TFT_ORANGE);
        s_canvas.println("WIFI");
        set_text(TFT_WHITE);
        s_canvas.println("Connecting...");
    } else {
        set_text(TFT_RED);
        s_canvas.println("NO WIFI");
        set_text(TFT_WHITE);
        s_canvas.println("Hold button");
        s_canvas.println("for pairing");
    }

    if (st->message[0]) {
        s_canvas.setCursor(0, 96);
        set_text(TFT_DARKGREY);
        print_clipped(st->message);
    }

    s_canvas.setCursor(0, 116);
    set_text(TFT_DARKGREY);
    s_canvas.println("Hold: Pair");
    display_present_canvas(s_canvas);
}

void display_app_show_pairing_hold(uint32_t held_ms, uint32_t target_ms)
{
    if (!s_inited) {
        return;
    }
    s_canvas.fillScreen(TFT_BLACK);
    s_canvas.setCursor(0, 0);
    set_text(TFT_YELLOW);
    s_canvas.println("PAIRING");
    set_text(TFT_WHITE);
    s_canvas.println("Keep holding");
    s_canvas.printf("%lu/%lu ms\n", (unsigned long)held_ms, (unsigned long)target_ms);
    s_canvas.println("Release cancel");
    display_present_canvas(s_canvas);
}

void display_app_show_error(const char *line1, const char *line2)
{
    if (!s_inited) {
        return;
    }
    s_canvas.fillScreen(TFT_BLACK);
    s_canvas.setCursor(0, 0);
    set_text(TFT_RED);
    s_canvas.println("ERROR");
    set_text(TFT_WHITE);
    if (line1) {
        s_canvas.println(line1);
    }
    if (line2) {
        s_canvas.println(line2);
    }
    display_present_canvas(s_canvas);
}
