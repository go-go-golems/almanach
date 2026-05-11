/*
 * provisioning_cmd.c — Console commands for BLE WiFi provisioning lifecycle.
 */

#include "provisioning_cmd.h"

#include <stdio.h>

#include "esp_console.h"
#include "esp_err.h"
#include "esp_log.h"
#include "esp_system.h"

#include "nvs_store.h"
#include "provisioning_mgr.h"
#include "wifi_mgr.h"

static const char *TAG __attribute__((unused)) = "provisioning_cmd";

static int do_prov_status(int argc, char **argv)
{
    (void)argc;
    (void)argv;

    provisioning_status_t st = {0};
    esp_err_t err = provisioning_mgr_get_status(&st);
    if (err != ESP_OK) {
        printf("Provisioning status failed: %s\n", esp_err_to_name(err));
        return 1;
    }

    printf("Provisioning manager:\n");
    printf("  initialized      : %s\n", st.initialized ? "yes" : "no");
    printf("  provisioned      : %s\n", st.provisioned ? "yes" : "no");
    printf("  BLE running      : %s\n", st.running ? "yes" : "no");
    printf("  client connected : %s\n", st.client_connected ? "yes" : "no");
    printf("  security ok      : %s\n", st.security_ok ? "yes" : "no");
    printf("  service name     : %s\n", st.service_name[0] ? st.service_name : "(not generated)");
    printf("  PoP              : %s\n", st.pop[0] ? st.pop : "(not generated)");

    if (wifi_mgr_is_connected()) {
        char ip[16] = {0};
        if (wifi_mgr_get_ip(ip, sizeof(ip)) == ESP_OK) {
            printf("WiFi: CONNECTED  IP: %s\n", ip);
        } else {
            printf("WiFi: CONNECTED  IP: (unavailable)\n");
        }
    } else {
        printf("WiFi: DISCONNECTED\n");
    }

    return 0;
}

static int do_prov_start(int argc, char **argv)
{
    (void)argc;
    (void)argv;

    bool started = false;
    esp_err_t err = provisioning_mgr_start_if_needed(&started);
    if (err != ESP_OK) {
        printf("BLE provisioning start failed: %s\n", esp_err_to_name(err));
        return 1;
    }

    if (started) {
        provisioning_status_t st = {0};
        provisioning_mgr_get_status(&st);
        printf("BLE provisioning started. Device: %s  PoP: %s\n", st.service_name, st.pop);
    } else {
        printf("BLE provisioning not started: device is already provisioned or already running.\n");
    }
    return 0;
}

static int do_prov_reset(int argc, char **argv)
{
    (void)argc;
    (void)argv;

    printf("Resetting WiFi/provisioning state...\n");
    wifi_mgr_disconnect();

    esp_err_t err = nvs_store_erase_wifi();
    if (err != ESP_OK) {
        printf("Warning: explicit WiFi credential erase failed: %s\n", esp_err_to_name(err));
    }

    err = provisioning_mgr_reset();
    if (err != ESP_OK) {
        printf("Provisioning reset failed: %s\n", esp_err_to_name(err));
        return 1;
    }

    printf("Provisioning reset complete. Rebooting...\n");
    fflush(stdout);
    esp_restart();
    return 0;
}

static void reg(const char *name, const char *help, esp_console_cmd_func_t func)
{
    const esp_console_cmd_t cmd = {
        .command = name,
        .help = help,
        .func = func,
    };
    ESP_ERROR_CHECK(esp_console_cmd_register(&cmd));
}

void provisioning_cmd_register(void)
{
    reg("prov_status", "Show BLE WiFi provisioning status, service name, PoP, and current IP",
        do_prov_status);
    reg("prov_start", "Start BLE WiFi provisioning if the device is not already provisioned",
        do_prov_start);
    reg("prov_reset", "Erase console/provisioned WiFi state and reboot for fresh BLE provisioning",
        do_prov_reset);
}
