#pragma once

#include <stdbool.h>
#include "esp_err.h"

#ifdef __cplusplus
extern "C" {
#endif

typedef struct {
    bool initialized;
    bool provisioned;
    bool running;
    bool client_connected;
    bool security_ok;
    char service_name[32];
    char pop[32];
} provisioning_status_t;

esp_err_t provisioning_mgr_init(void);
esp_err_t provisioning_mgr_is_provisioned(bool *out_provisioned);
esp_err_t provisioning_mgr_start_if_needed(bool *out_started);
esp_err_t provisioning_mgr_start_force(void);
esp_err_t provisioning_mgr_stop(void);
esp_err_t provisioning_mgr_reset(void);
esp_err_t provisioning_mgr_get_status(provisioning_status_t *out);

#ifdef __cplusplus
}
#endif
