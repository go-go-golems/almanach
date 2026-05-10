#pragma once

#include <stdbool.h>
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    bool wifi_connected;
    char ip[16];
    bool provisioned;
    bool provisioning_running;
    bool provisioning_client_connected;
    bool provisioning_security_ok;
    char service_name[32];
    char pop[32];
    char message[64];
} display_status_t;

esp_err_t display_app_init(void);
bool display_app_is_ready(void);
void display_app_show_boot(const char *line1, const char *line2);
void display_app_show_status(const display_status_t *status);
void display_app_show_pairing_hold(uint32_t held_ms, uint32_t target_ms);
void display_app_show_error(const char *line1, const char *line2);

#ifdef __cplusplus
}
#endif
