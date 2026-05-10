/*
 * SToMS3R — AtomS3R Lite Thermal Printer Console Firmware
 */

#include <stdio.h>
#include <string.h>

#include "esp_console.h"
#include "esp_event.h"
#include "esp_log.h"
#include "esp_netif.h"
#include "esp_wifi.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "esp_check.h"
#include "nvs_flash.h"

#include "display_app.h"
#include "nvs_store.h"
#include "printer_cmd.h"
#include "printer_drv.h"
#include "provisioning_cmd.h"
#include "provisioning_mgr.h"
#include "web_server.h"
#include "wifi_cmd.h"
#include "wifi_mgr.h"

static const char *TAG = "stoms3r";

static bool app_supported_baud(int rate)
{
    switch (rate) {
        case 9600:
        case 19200:
        case 38400:
        case 57600:
        case 115200:
        case 230400:
        case 460800:
        case 921600:
            return true;
        default:
            return false;
    }
}

static bool app_supported_speed(int speed)
{
    static const int speeds[] = { 25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220 };
    for (size_t i = 0; i < sizeof(speeds) / sizeof(speeds[0]); i++) {
        if (speeds[i] == speed) return true;
    }
    return false;
}

static esp_err_t apply_saved_printer_settings(void)
{
    printer_settings_t settings;
    esp_err_t err = nvs_store_load_printer_settings(&settings);
    if (err == ESP_ERR_NVS_NOT_FOUND) {
        ESP_LOGI(TAG, "No saved printer settings");
        return ESP_OK;
    }
    if (err != ESP_OK) {
        ESP_LOGW(TAG, "Could not load printer settings: %s", esp_err_to_name(err));
        return err;
    }

    if (!app_supported_baud(settings.baud) || settings.density < 0 || settings.density > 39 ||
        !app_supported_speed(settings.speed) ||
        (settings.graphics_mode != 30 && settings.graphics_mode != 31 && settings.graphics_mode != 32)) {
        ESP_LOGW(TAG, "Ignoring invalid saved printer settings: baud=%ld density=%ld speed=%ld graphics_mode=%ld",
                 (long)settings.baud, (long)settings.density,
                 (long)settings.speed, (long)settings.graphics_mode);
        return ESP_ERR_INVALID_STATE;
    }

    ESP_LOGI(TAG, "Applying saved printer settings: baud=%ld density=%ld speed=%ld graphics_mode=%ld",
             (long)settings.baud, (long)settings.density,
             (long)settings.speed, (long)settings.graphics_mode);

    /* At boot we set the ESP32 UART side directly. This assumes the printer-side
     * baud setting was persisted by the K118 after a prior set_baudrate command.
     * If the printer was power-cycled back to 9600, recover from the console with
     * printer_baud 9600 or clear the saved settings. */
    ESP_RETURN_ON_ERROR(printer_drv_set_baud(settings.baud), TAG, "set saved UART baud");
    ESP_RETURN_ON_ERROR(printer_drv_set_density((uint8_t)settings.density), TAG, "set saved density");
    ESP_RETURN_ON_ERROR(printer_drv_set_speed((uint8_t)settings.speed), TAG, "set saved speed");
    ESP_RETURN_ON_ERROR(printer_drv_set_graphics_mode((uint8_t)settings.graphics_mode), TAG, "set saved graphics mode");

    return ESP_OK;
}

static void start_network_onboarding(void)
{
    ESP_ERROR_CHECK(provisioning_mgr_init());

    bool provisioned = false;
    esp_err_t err = provisioning_mgr_is_provisioned(&provisioned);
    if (err == ESP_OK && provisioned) {
        ESP_LOGI(TAG, "Provisioned WiFi found — starting station mode");
        ESP_ERROR_CHECK(wifi_mgr_start_station());
        return;
    } else if (err != ESP_OK) {
        ESP_LOGW(TAG, "Could not query provisioning state: %s", esp_err_to_name(err));
    }

    char ssid[64] = {0};
    char password[64] = {0};
    if (nvs_store_load_wifi(ssid, sizeof(ssid),
                             password, sizeof(password)) == ESP_OK) {
        ESP_LOGI(TAG, "Console-saved WiFi found: \"%s\" — connecting...", ssid);
        wifi_mgr_connect(ssid, password);
        return;
    }

    ESP_LOGI(TAG, "No saved WiFi credentials — starting BLE provisioning");
    bool started = false;
    err = provisioning_mgr_start_if_needed(&started);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "BLE provisioning start failed: %s", esp_err_to_name(err));
    } else if (!started) {
        ESP_LOGI(TAG, "BLE provisioning was not needed");
    }
}

/* Background task: wait for WiFi, then start web server */
static void web_server_task(void *arg)
{
    (void)arg;
    /* Poll until WiFi is connected (max 30 s) */
    for (int i = 0; i < 60; i++) {
        if (wifi_mgr_is_connected()) {
            ESP_LOGI(TAG, "WiFi connected — starting web server");
            web_server_start();
            vTaskDelete(NULL);
            return;
        }
        vTaskDelay(pdMS_TO_TICKS(500));
    }
    ESP_LOGW(TAG, "WiFi not connected after 30s — web server not started");
    vTaskDelete(NULL);
}

void app_main(void)
{
    ESP_LOGI(TAG, "============================");
    ESP_LOGI(TAG, "SToMS3R starting...");
    ESP_LOGI(TAG, "AtomS3R Lite + K118 printer");
    ESP_LOGI(TAG, "============================");

    /* 1. Local display (best-effort; serial console remains authoritative). */
    esp_err_t display_err = display_app_init();
    if (display_err == ESP_OK) {
        display_app_show_boot("AtomS3R", "Booting...");
    } else {
        ESP_LOGW(TAG, "Display init skipped/failed: %s", esp_err_to_name(display_err));
    }

    /* 2. NVS */
    ESP_ERROR_CHECK(nvs_store_init());

    /* 3. Network stack */
    ESP_ERROR_CHECK(esp_netif_init());
    ESP_ERROR_CHECK(esp_event_loop_create_default());

    /* 4. WiFi manager */
    ESP_ERROR_CHECK(wifi_mgr_init());

    /* 5. Printer UART */
    ESP_ERROR_CHECK(printer_drv_init());
    apply_saved_printer_settings();

    /* 6. Start WiFi from saved credentials or BLE provisioning */
    start_network_onboarding();

    /* 7. Start background task that launches the web server once WiFi is up */
    xTaskCreate(web_server_task, "web_wait", 4096, NULL, 2, NULL);

    /* 8. Start the interactive console (blocks forever) */
    esp_console_repl_t *repl = NULL;
    esp_console_repl_config_t repl_cfg = ESP_CONSOLE_REPL_CONFIG_DEFAULT();
    repl_cfg.prompt = "stoms3r> ";
    repl_cfg.max_cmdline_length = 256;
    repl_cfg.task_stack_size = 6144;

    esp_console_dev_usb_serial_jtag_config_t hw_cfg =
        ESP_CONSOLE_DEV_USB_SERIAL_JTAG_CONFIG_DEFAULT();

    ESP_ERROR_CHECK(
        esp_console_new_repl_usb_serial_jtag(&hw_cfg, &repl_cfg, &repl));

    /* 9. Register commands */
    esp_console_register_help_command();
    printer_cmd_register();
    wifi_cmd_register();
    provisioning_cmd_register();

    /* 10. Start REPL — does not return */
    ESP_ERROR_CHECK(esp_console_start_repl(repl));
}
