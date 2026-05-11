#pragma once

#include "esp_err.h"

/**
 * Start the HTTP server on port 80.
 * Serves index.html at / and provides /api/ endpoints.
 * Should be called after WiFi is connected.
 */
esp_err_t web_server_start(void);

/**
 * Stop the HTTP server.
 */
void web_server_stop(void);
