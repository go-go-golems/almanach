#pragma once

#include <stdbool.h>
#include "esp_err.h"

/**
 * Initialize the WiFi manager: init netif, create default STA, register
 * event handlers. Must be called once at startup.
 */
esp_err_t wifi_mgr_init(void);

/**
 * Perform a blocking Wi-Fi scan and print the results as a table.
 */
esp_err_t wifi_mgr_scan(void);

/**
 * Connect to a Wi-Fi access point. If successful, credentials are NOT
 * saved automatically — call nvs_store_save_wifi() explicitly.
 */
esp_err_t wifi_mgr_connect(const char *ssid, const char *password);

/**
 * Disconnect from the current Wi-Fi network and stop the station.
 */
esp_err_t wifi_mgr_disconnect(void);

/**
 * Returns true if currently connected to Wi-Fi with an IP address.
 */
bool wifi_mgr_is_connected(void);

/**
 * Copy the current IP address string into `buf` (e.g. "192.168.1.42").
 * Returns ESP_OK on success, ESP_FAIL if not connected.
 */
esp_err_t wifi_mgr_get_ip(char *buf, size_t buf_len);
