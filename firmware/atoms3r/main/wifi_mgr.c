/*
 * wifi_mgr.c — WiFi station manager for SToMS3R.
 *
 * Provides init/scan/connect/disconnect with auto-reconnect via the
 * ESP event loop.  The manager does NOT persist credentials — that is
 * handled by nvs_store.
 */

#include "wifi_mgr.h"

#include <string.h>
#include "esp_check.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/event_groups.h"

static const char *TAG = "wifi_mgr";

static const int CONNECTED_BIT = BIT0;

static EventGroupHandle_t s_eg;
static esp_netif_t *s_sta_netif = NULL;
static bool s_initialized = false;

/* ---- internal event handler ------------------------------------------- */

static void event_handler(void *arg, esp_event_base_t base,
                           int32_t id, void *data)
{
    (void)arg;

    if (base == WIFI_EVENT) {
        switch (id) {
        case WIFI_EVENT_STA_START:
            ESP_LOGI(TAG, "STA started, connecting...");
            esp_wifi_connect();
            break;
        case WIFI_EVENT_STA_CONNECTED:
            ESP_LOGI(TAG, "STA associated with AP");
            break;
        case WIFI_EVENT_STA_DISCONNECTED: {
            const wifi_event_sta_disconnected_t *evt =
                (const wifi_event_sta_disconnected_t *)data;
            ESP_LOGW(TAG, "STA disconnected: reason=%d", evt->reason);
            xEventGroupClearBits(s_eg, CONNECTED_BIT);
            /* Auto-reconnect */
            esp_wifi_connect();
            break;
        }
        default:
            break;
        }
    } else if (base == IP_EVENT && id == IP_EVENT_STA_GOT_IP) {
        const ip_event_got_ip_t *evt = (const ip_event_got_ip_t *)data;
        ESP_LOGI(TAG, "Got IP: " IPSTR, IP2STR(&evt->ip_info.ip));
        xEventGroupSetBits(s_eg, CONNECTED_BIT);
    }
}

/* ---- public API ------------------------------------------------------- */

esp_err_t wifi_mgr_init(void)
{
    if (s_initialized) return ESP_OK;

    s_eg = xEventGroupCreate();
    if (!s_eg) return ESP_ERR_NO_MEM;

    s_sta_netif = esp_netif_create_default_wifi_sta();

    wifi_init_config_t cfg = WIFI_INIT_CONFIG_DEFAULT();
    esp_err_t err = esp_wifi_init(&cfg);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "wifi init failed: %s", esp_err_to_name(err));
        return err;
    }

    ESP_ERROR_CHECK(
        esp_event_handler_instance_register(WIFI_EVENT, ESP_EVENT_ANY_ID,
                                             &event_handler, NULL, NULL));
    ESP_ERROR_CHECK(
        esp_event_handler_instance_register(IP_EVENT, IP_EVENT_STA_GOT_IP,
                                             &event_handler, NULL, NULL));

    s_initialized = true;
    ESP_LOGI(TAG, "WiFi manager initialized");
    return ESP_OK;
}

esp_err_t wifi_mgr_scan(void)
{
    esp_err_t err = esp_wifi_set_mode(WIFI_MODE_STA);
    if (err != ESP_OK) return err;
    err = esp_wifi_start();
    if (err != ESP_OK) return err;

    wifi_scan_config_t scan_cfg = {
        .ssid        = NULL,
        .bssid       = NULL,
        .channel     = 0,
        .show_hidden = false,
    };

    err = esp_wifi_scan_start(&scan_cfg, true);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "scan failed: %s", esp_err_to_name(err));
        esp_wifi_stop();
        return err;
    }

    uint16_t ap_count = 0;
    esp_wifi_scan_get_ap_num(&ap_count);
    if (ap_count == 0) {
        printf("No access points found\n");
        esp_wifi_stop();
        return ESP_OK;
    }

    wifi_ap_record_t *aps = (wifi_ap_record_t *)malloc(ap_count * sizeof(wifi_ap_record_t));
    if (!aps) {
        esp_wifi_stop();
        return ESP_ERR_NO_MEM;
    }

    ESP_ERROR_CHECK(esp_wifi_scan_get_ap_records(&ap_count, aps));

    /* Print table */
    printf("%4s %-32s %5s %3s %s\n",
           "#", "SSID", "RSSI", "Ch", "Auth");
    printf("---- -------------------------------- ----- --- -----\n");
    for (uint16_t i = 0; i < ap_count; i++) {
        const char *auth;
        switch (aps[i].authmode) {
        case WIFI_AUTH_OPEN:          auth = "OPEN";    break;
        case WIFI_AUTH_WEP:           auth = "WEP";     break;
        case WIFI_AUTH_WPA_PSK:       auth = "WPA";     break;
        case WIFI_AUTH_WPA2_PSK:      auth = "WPA2";    break;
        case WIFI_AUTH_WPA_WPA2_PSK:  auth = "WPA/WPA2"; break;
        case WIFI_AUTH_WPA3_PSK:      auth = "WPA3";    break;
        default:                      auth = "???";     break;
        }
        printf("%3u  %-32s %4d  %3u %s\n",
               i + 1,
               aps[i].ssid,
               aps[i].rssi,
               aps[i].primary,
               auth);
    }
    printf("(%u APs found)\n", ap_count);

    free(aps);
    esp_wifi_stop();
    return ESP_OK;
}

esp_err_t wifi_mgr_connect(const char *ssid, const char *password)
{
    esp_err_t err = esp_wifi_set_mode(WIFI_MODE_STA);
    if (err != ESP_OK) return err;
    err = esp_wifi_start();
    if (err != ESP_OK) return err;

    wifi_config_t wifi_config = {0};
    strncpy((char *)wifi_config.sta.ssid, ssid, sizeof(wifi_config.sta.ssid) - 1);
    strncpy((char *)wifi_config.sta.password, password, sizeof(wifi_config.sta.password) - 1);
    wifi_config.sta.threshold.authmode = WIFI_AUTH_WPA2_PSK;

    err = esp_wifi_set_config(WIFI_IF_STA, &wifi_config);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "set config failed: %s", esp_err_to_name(err));
        return err;
    }

    /* esp_wifi_connect() will be called automatically by the STA_START
     * event handler.  Wait for either CONNECTED_BIT or a 15 s timeout. */
    EventBits_t bits = xEventGroupWaitBits(s_eg, CONNECTED_BIT,
                                            pdFALSE, pdTRUE,
                                            pdMS_TO_TICKS(15000));
    if (bits & CONNECTED_BIT) {
        return ESP_OK;
    }

    ESP_LOGW(TAG, "WiFi connect timed out (15 s)");
    return ESP_ERR_TIMEOUT;
}

esp_err_t wifi_mgr_disconnect(void)
{
    xEventGroupClearBits(s_eg, CONNECTED_BIT);
    esp_wifi_disconnect();
    esp_wifi_stop();
    ESP_LOGI(TAG, "WiFi disconnected and stopped");
    return ESP_OK;
}

bool wifi_mgr_is_connected(void)
{
    return (xEventGroupGetBits(s_eg) & CONNECTED_BIT) != 0;
}

esp_err_t wifi_mgr_get_ip(char *buf, size_t buf_len)
{
    if (!wifi_mgr_is_connected()) return ESP_FAIL;

    esp_netif_ip_info_t ip_info;
    if (esp_netif_get_ip_info(s_sta_netif, &ip_info) != ESP_OK) {
        return ESP_FAIL;
    }
    snprintf(buf, buf_len, IPSTR, IP2STR(&ip_info.ip));
    return ESP_OK;
}
