#include "provisioning_mgr.h"

#include <stdio.h>
#include <string.h>

#include "esp_check.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_mac.h"
#include "esp_wifi.h"
#include "protocomm_security.h"
#include "protocomm_ble.h"
#include "wifi_provisioning/manager.h"
#include "wifi_provisioning/scheme_ble.h"

static const char *TAG = "provisioning";

static bool s_initialized = false;
static bool s_running = false;
static bool s_client_connected = false;
static bool s_security_ok = false;
static bool s_handlers_registered = false;
static char s_service_name[32] = {0};
static char s_pop[32] = {0};

static void make_service_identity(void)
{
    uint8_t mac[6] = {0};
    esp_err_t err = esp_read_mac(mac, ESP_MAC_WIFI_STA);
    if (err != ESP_OK) {
        ESP_LOGW(TAG, "esp_read_mac failed: %s; using fallback identity", esp_err_to_name(err));
        strlcpy(s_service_name, "ALM_SETUP", sizeof(s_service_name));
        strlcpy(s_pop, "alm-setup", sizeof(s_pop));
        return;
    }

    snprintf(s_service_name, sizeof(s_service_name),
             "ALM_%02X%02X%02X", mac[3], mac[4], mac[5]);
    snprintf(s_pop, sizeof(s_pop),
             "alm-%02x%02x%02x", mac[3], mac[4], mac[5]);
}

static void log_provisioning_payload(void)
{
    ESP_LOGI(TAG, "Provision with Espressif's ESP BLE Provisioning app or compatible client:");
    ESP_LOGI(TAG, "  Transport : BLE");
    ESP_LOGI(TAG, "  Device    : %s", s_service_name);
    ESP_LOGI(TAG, "  Security  : Security 1");
    ESP_LOGI(TAG, "  PoP       : %s", s_pop);
    ESP_LOGI(TAG, "  QR data   : {\"ver\":\"v1\",\"name\":\"%s\",\"pop\":\"%s\",\"transport\":\"ble\"}",
             s_service_name, s_pop);
}

static void provisioning_event_handler(void *arg, esp_event_base_t event_base,
                                       int32_t event_id, void *event_data)
{
    (void)arg;

    if (event_base == WIFI_PROV_EVENT) {
        switch (event_id) {
        case WIFI_PROV_INIT:
            ESP_LOGI(TAG, "WiFi provisioning manager initialized");
            break;
        case WIFI_PROV_START:
            s_running = true;
            ESP_LOGI(TAG, "BLE WiFi provisioning started");
            break;
        case WIFI_PROV_CRED_RECV: {
            const wifi_sta_config_t *cfg = (const wifi_sta_config_t *)event_data;
            ESP_LOGI(TAG, "Received WiFi credentials for SSID '%s'", (const char *)cfg->ssid);
            break;
        }
        case WIFI_PROV_CRED_FAIL: {
            const wifi_prov_sta_fail_reason_t *reason =
                (const wifi_prov_sta_fail_reason_t *)event_data;
            ESP_LOGE(TAG, "Provisioned WiFi connection failed: %s",
                     (*reason == WIFI_PROV_STA_AUTH_ERROR) ? "authentication failed" : "AP not found");
            break;
        }
        case WIFI_PROV_CRED_SUCCESS:
            ESP_LOGI(TAG, "Provisioned WiFi credentials connected successfully");
            break;
        case WIFI_PROV_END:
            s_running = false;
            ESP_LOGI(TAG, "BLE WiFi provisioning ended");
            wifi_prov_mgr_deinit();
            s_initialized = false;
            break;
        case WIFI_PROV_DEINIT:
            s_running = false;
            s_initialized = false;
            ESP_LOGI(TAG, "WiFi provisioning manager deinitialized");
            break;
        default:
            break;
        }
        return;
    }

    if (event_base == PROTOCOMM_TRANSPORT_BLE_EVENT) {
        switch (event_id) {
        case PROTOCOMM_TRANSPORT_BLE_CONNECTED:
            s_client_connected = true;
            ESP_LOGI(TAG, "BLE provisioning client connected");
            break;
        case PROTOCOMM_TRANSPORT_BLE_DISCONNECTED:
            s_client_connected = false;
            ESP_LOGI(TAG, "BLE provisioning client disconnected");
            break;
        default:
            break;
        }
        return;
    }

    if (event_base == PROTOCOMM_SECURITY_SESSION_EVENT) {
        switch (event_id) {
        case PROTOCOMM_SECURITY_SESSION_SETUP_OK:
            s_security_ok = true;
            ESP_LOGI(TAG, "Provisioning security session established");
            break;
        case PROTOCOMM_SECURITY_SESSION_CREDENTIALS_MISMATCH:
            s_security_ok = false;
            ESP_LOGE(TAG, "Provisioning PoP mismatch");
            break;
        case PROTOCOMM_SECURITY_SESSION_INVALID_SECURITY_PARAMS:
            s_security_ok = false;
            ESP_LOGE(TAG, "Provisioning security parameters invalid");
            break;
        default:
            break;
        }
    }
}

static esp_err_t register_event_handlers(void)
{
    if (s_handlers_registered) {
        return ESP_OK;
    }

    ESP_RETURN_ON_ERROR(
        esp_event_handler_register(WIFI_PROV_EVENT, ESP_EVENT_ANY_ID,
                                   provisioning_event_handler, NULL),
        TAG, "register WIFI_PROV_EVENT handler");
    ESP_RETURN_ON_ERROR(
        esp_event_handler_register(PROTOCOMM_TRANSPORT_BLE_EVENT, ESP_EVENT_ANY_ID,
                                   provisioning_event_handler, NULL),
        TAG, "register PROTOCOMM_TRANSPORT_BLE_EVENT handler");
    ESP_RETURN_ON_ERROR(
        esp_event_handler_register(PROTOCOMM_SECURITY_SESSION_EVENT, ESP_EVENT_ANY_ID,
                                   provisioning_event_handler, NULL),
        TAG, "register PROTOCOMM_SECURITY_SESSION_EVENT handler");
    s_handlers_registered = true;
    return ESP_OK;
}

esp_err_t provisioning_mgr_init(void)
{
    if (s_initialized) {
        return ESP_OK;
    }

    make_service_identity();

    wifi_prov_mgr_config_t config = {
        .scheme = wifi_prov_scheme_ble,
        .scheme_event_handler = WIFI_PROV_SCHEME_BLE_EVENT_HANDLER_FREE_BTDM,
        .app_event_handler = WIFI_PROV_EVENT_HANDLER_NONE,
    };

    ESP_RETURN_ON_ERROR(wifi_prov_mgr_init(config), TAG, "wifi_prov_mgr_init");

    uint8_t custom_service_uuid[] = {
        0xb4, 0xdf, 0x5a, 0x1c, 0x3f, 0x6b, 0xf4, 0xbf,
        0xea, 0x4a, 0x82, 0x03, 0x04, 0x90, 0x1a, 0x02,
    };
    wifi_prov_scheme_ble_set_service_uuid(custom_service_uuid);

    ESP_RETURN_ON_ERROR(register_event_handlers(), TAG, "register provisioning handlers");

    s_initialized = true;
    s_running = false;
    s_client_connected = false;
    s_security_ok = false;
    return ESP_OK;
}

esp_err_t provisioning_mgr_is_provisioned(bool *out_provisioned)
{
    if (out_provisioned == NULL) {
        return ESP_ERR_INVALID_ARG;
    }
    ESP_RETURN_ON_ERROR(provisioning_mgr_init(), TAG, "ensure provisioning manager initialized");
    return wifi_prov_mgr_is_provisioned(out_provisioned);
}

esp_err_t provisioning_mgr_start_if_needed(bool *out_started)
{
    if (out_started != NULL) {
        *out_started = false;
    }

    ESP_RETURN_ON_ERROR(provisioning_mgr_init(), TAG, "ensure provisioning manager initialized");

    if (s_running) {
        ESP_LOGI(TAG, "BLE provisioning already running");
        return ESP_OK;
    }

    bool provisioned = false;
    ESP_RETURN_ON_ERROR(wifi_prov_mgr_is_provisioned(&provisioned), TAG, "check provisioned state");
    if (provisioned) {
        ESP_LOGI(TAG, "Device already provisioned; BLE provisioning not started");
        return ESP_OK;
    }

    ESP_RETURN_ON_ERROR(provisioning_mgr_start_force(), TAG, "start provisioning");
    if (out_started != NULL) {
        *out_started = true;
    }
    return ESP_OK;
}

esp_err_t provisioning_mgr_start_force(void)
{
    ESP_RETURN_ON_ERROR(provisioning_mgr_init(), TAG, "ensure provisioning manager initialized");

    if (s_running) {
        ESP_LOGI(TAG, "BLE provisioning already running");
        return ESP_OK;
    }

    const wifi_prov_security_t security = WIFI_PROV_SECURITY_1;
    const wifi_prov_security1_params_t *security_params = s_pop;
    const char *service_key = NULL;

    ESP_RETURN_ON_ERROR(
        wifi_prov_mgr_start_provisioning(security, (const void *)security_params,
                                         s_service_name, service_key),
        TAG, "wifi_prov_mgr_start_provisioning");

    s_running = true;
    log_provisioning_payload();
    return ESP_OK;
}

esp_err_t provisioning_mgr_stop(void)
{
    if (!s_initialized) {
        return ESP_OK;
    }
    if (s_running) {
        wifi_prov_mgr_stop_provisioning();
        s_running = false;
    }
    wifi_prov_mgr_deinit();
    s_initialized = false;
    return ESP_OK;
}

esp_err_t provisioning_mgr_reset(void)
{
    ESP_RETURN_ON_ERROR(provisioning_mgr_init(), TAG, "ensure provisioning manager initialized");
    ESP_RETURN_ON_ERROR(wifi_prov_mgr_reset_provisioning(), TAG, "reset provisioning state");
    s_running = false;
    s_client_connected = false;
    s_security_ok = false;
    ESP_LOGW(TAG, "WiFi provisioning state reset");
    return ESP_OK;
}

esp_err_t provisioning_mgr_get_status(provisioning_status_t *out)
{
    if (out == NULL) {
        return ESP_ERR_INVALID_ARG;
    }

    memset(out, 0, sizeof(*out));
    out->initialized = s_initialized;
    out->running = s_running;
    out->client_connected = s_client_connected;
    out->security_ok = s_security_ok;
    strlcpy(out->service_name, s_service_name, sizeof(out->service_name));
    strlcpy(out->pop, s_pop, sizeof(out->pop));

    if (s_initialized) {
        bool provisioned = false;
        if (wifi_prov_mgr_is_provisioned(&provisioned) == ESP_OK) {
            out->provisioned = provisioned;
        }
    }

    return ESP_OK;
}
