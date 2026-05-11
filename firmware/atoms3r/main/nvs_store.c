/*
 * nvs_store.c — Thin NVS wrapper for WiFi credential persistence.
 */

#include "nvs_store.h"

#include <string.h>
#include "esp_log.h"
#include "nvs.h"
#include "nvs_flash.h"

static const char *TAG = "nvs_store";

esp_err_t nvs_store_init(void)
{
    esp_err_t ret = nvs_flash_init();
    if (ret == ESP_ERR_NVS_NO_FREE_PAGES ||
        ret == ESP_ERR_NVS_NEW_VERSION_FOUND) {
        ESP_LOGW(TAG, "NVS init returned %s — erasing and retrying",
                 esp_err_to_name(ret));
        ESP_ERROR_CHECK(nvs_flash_erase());
        ret = nvs_flash_init();
    }
    if (ret == ESP_OK) {
        ESP_LOGI(TAG, "NVS initialized");
    }
    return ret;
}

esp_err_t nvs_store_save_wifi(const char *ssid, const char *password)
{
    nvs_handle_t handle;
    esp_err_t err = nvs_open("wifi", NVS_READWRITE, &handle);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "nvs_open failed: %s", esp_err_to_name(err));
        return err;
    }

    err = nvs_set_str(handle, "ssid", ssid);
    if (err != ESP_OK) {
        nvs_close(handle);
        return err;
    }

    err = nvs_set_str(handle, "password", password);
    if (err != ESP_OK) {
        nvs_close(handle);
        return err;
    }

    err = nvs_commit(handle);
    nvs_close(handle);

    if (err == ESP_OK) {
        ESP_LOGI(TAG, "WiFi credentials saved: \"%s\"", ssid);
    }
    return err;
}

esp_err_t nvs_store_load_wifi(char *ssid, size_t ssid_len,
                               char *password, size_t password_len)
{
    nvs_handle_t handle;
    esp_err_t err = nvs_open("wifi", NVS_READONLY, &handle);
    if (err != ESP_OK) {
        return err; /* Namespace doesn't exist yet — no saved WiFi */
    }

    err = nvs_get_str(handle, "ssid", ssid, &ssid_len);
    if (err != ESP_OK) {
        nvs_close(handle);
        return err;
    }

    err = nvs_get_str(handle, "password", password, &password_len);
    nvs_close(handle);
    return err;
}

esp_err_t nvs_store_erase_wifi(void)
{
    nvs_handle_t handle;
    esp_err_t err = nvs_open("wifi", NVS_READWRITE, &handle);
    if (err != ESP_OK) {
        return err;
    }

    /* Ignore errors — keys may not exist */
    nvs_erase_key(handle, "ssid");
    nvs_erase_key(handle, "password");
    nvs_commit(handle);
    nvs_close(handle);

    ESP_LOGI(TAG, "WiFi credentials erased");
    return ESP_OK;
}

esp_err_t nvs_store_save_printer_settings(const printer_settings_t *settings)
{
    if (settings == NULL) {
        return ESP_ERR_INVALID_ARG;
    }

    nvs_handle_t handle;
    esp_err_t err = nvs_open("printer", NVS_READWRITE, &handle);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "nvs_open printer failed: %s", esp_err_to_name(err));
        return err;
    }

    err = nvs_set_i32(handle, "baud", settings->baud);
    if (err == ESP_OK) err = nvs_set_i32(handle, "density", settings->density);
    if (err == ESP_OK) err = nvs_set_i32(handle, "speed", settings->speed);
    if (err == ESP_OK) err = nvs_set_i32(handle, "gmode", settings->graphics_mode);
    if (err == ESP_OK) err = nvs_commit(handle);

    nvs_close(handle);

    if (err == ESP_OK) {
        ESP_LOGI(TAG, "Printer settings saved: baud=%ld density=%ld speed=%ld graphics_mode=%ld",
                 (long)settings->baud, (long)settings->density,
                 (long)settings->speed, (long)settings->graphics_mode);
    }
    return err;
}

esp_err_t nvs_store_load_printer_settings(printer_settings_t *settings)
{
    if (settings == NULL) {
        return ESP_ERR_INVALID_ARG;
    }

    nvs_handle_t handle;
    esp_err_t err = nvs_open("printer", NVS_READONLY, &handle);
    if (err != ESP_OK) {
        return err;
    }

    err = nvs_get_i32(handle, "baud", &settings->baud);
    if (err == ESP_OK) err = nvs_get_i32(handle, "density", &settings->density);
    if (err == ESP_OK) err = nvs_get_i32(handle, "speed", &settings->speed);
    if (err == ESP_OK) err = nvs_get_i32(handle, "gmode", &settings->graphics_mode);

    nvs_close(handle);
    return err;
}

esp_err_t nvs_store_erase_printer_settings(void)
{
    nvs_handle_t handle;
    esp_err_t err = nvs_open("printer", NVS_READWRITE, &handle);
    if (err != ESP_OK) {
        if (err == ESP_ERR_NVS_NOT_FOUND) {
            ESP_LOGI(TAG, "No printer settings to erase");
            return ESP_OK;
        }
        return err;
    }

    nvs_erase_key(handle, "baud");
    nvs_erase_key(handle, "density");
    nvs_erase_key(handle, "speed");
    nvs_erase_key(handle, "gmode");
    nvs_commit(handle);
    nvs_close(handle);

    ESP_LOGI(TAG, "Printer settings erased");
    return ESP_OK;
}
