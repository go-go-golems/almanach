/*
 * AtomS3R GC9107 display bring-up via M5GFX/LovyanGFX.
 */

#include "display_hal.h"

#include "sdkconfig.h"

#include "esp_log.h"

#include "M5GFX.h"
#include "lgfx/v1/panel/Panel_GC9A01.hpp"
#include "lgfx/v1/platforms/esp32/Bus_SPI.hpp"

static const char *TAG = "display_hal";

static constexpr int PIN_LCD_CS = 14;
static constexpr int PIN_LCD_SCK = 15;
static constexpr int PIN_LCD_MOSI = 21;
static constexpr int PIN_LCD_DC = 42;
static constexpr int PIN_LCD_RST = 48;

static lgfx::Bus_SPI s_bus;
static lgfx::Panel_GC9107 s_panel;
static m5gfx::M5GFX s_display;

m5gfx::M5GFX &display_get(void)
{
    return s_display;
}

bool display_init_m5gfx(void)
{
#if !CONFIG_ALMANACH_ATOMS3R_DISPLAY_ENABLE
    return false;
#else
    {
        auto cfg = s_bus.config();
        cfg.spi_host = SPI3_HOST;
        cfg.spi_mode = 0;
        cfg.spi_3wire = true;
        cfg.freq_write = CONFIG_ALMANACH_ATOMS3R_LCD_SPI_PCLK_HZ;
        cfg.freq_read = 16000000;
        cfg.pin_sclk = PIN_LCD_SCK;
        cfg.pin_mosi = PIN_LCD_MOSI;
        cfg.pin_miso = -1;
        cfg.pin_dc = PIN_LCD_DC;
        s_bus.config(cfg);
    }

    {
        s_panel.bus(&s_bus);
        auto pcfg = s_panel.config();
        pcfg.pin_cs = PIN_LCD_CS;
        pcfg.pin_rst = PIN_LCD_RST;
        pcfg.panel_width = CONFIG_ALMANACH_ATOMS3R_LCD_HRES;
        pcfg.panel_height = CONFIG_ALMANACH_ATOMS3R_LCD_VRES;
        pcfg.offset_x = CONFIG_ALMANACH_ATOMS3R_LCD_X_OFFSET;
        pcfg.offset_y = CONFIG_ALMANACH_ATOMS3R_LCD_Y_OFFSET;
        pcfg.readable = false;
#if CONFIG_ALMANACH_ATOMS3R_LCD_INVERT
        pcfg.invert = true;
#else
        pcfg.invert = false;
#endif
#if CONFIG_ALMANACH_ATOMS3R_LCD_RGB_ORDER_RGB
        pcfg.rgb_order = true;
#else
        pcfg.rgb_order = false;
#endif
        pcfg.bus_shared = false;
        s_panel.config(pcfg);
    }

    ESP_LOGI(TAG, "m5gfx init: pclk=%dHz gap=(%d,%d)",
             CONFIG_ALMANACH_ATOMS3R_LCD_SPI_PCLK_HZ,
             CONFIG_ALMANACH_ATOMS3R_LCD_X_OFFSET,
             CONFIG_ALMANACH_ATOMS3R_LCD_Y_OFFSET);

    bool ok = s_display.init(&s_panel);
    if (!ok) {
        ESP_LOGE(TAG, "M5GFX init failed");
        return false;
    }

    s_display.setColorDepth(16);
    s_display.setRotation(0);
    return true;
#endif
}

void display_present_canvas(M5Canvas &canvas)
{
#if CONFIG_ALMANACH_ATOMS3R_DISPLAY_ENABLE
    canvas.pushSprite(0, 0);
#if CONFIG_ALMANACH_ATOMS3R_PRESENT_USE_DMA
    s_display.waitDMA();
#endif
#else
    (void)canvas;
#endif
}
