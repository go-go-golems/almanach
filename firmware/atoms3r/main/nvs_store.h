#pragma once

#include <stdint.h>

#include "esp_err.h"

typedef struct {
    int32_t baud;
    int32_t density;
    int32_t speed;
    int32_t graphics_mode;
} printer_settings_t;

/**
 * Initialize NVS flash. Handles the "no free pages" case by erasing and
 * re-initializing. Must be called before any other nvs_store_* function.
 */
esp_err_t nvs_store_init(void);

/**
 * Save WiFi credentials to NVS (namespace "wifi").
 */
esp_err_t nvs_store_save_wifi(const char *ssid, const char *password);

/**
 * Load WiFi credentials from NVS. Returns ESP_ERR_NVS_NOT_FOUND if no
 * credentials have been saved.
 */
esp_err_t nvs_store_load_wifi(char *ssid, size_t ssid_len,
                               char *password, size_t password_len);

/**
 * Erase saved WiFi credentials.
 */
esp_err_t nvs_store_erase_wifi(void);

/**
 * Save printer startup settings to NVS (namespace "printer").
 */
esp_err_t nvs_store_save_printer_settings(const printer_settings_t *settings);

/**
 * Load printer startup settings from NVS. Returns ESP_ERR_NVS_NOT_FOUND if no
 * settings have been saved.
 */
esp_err_t nvs_store_load_printer_settings(printer_settings_t *settings);

/**
 * Erase saved printer startup settings.
 */
esp_err_t nvs_store_erase_printer_settings(void);
