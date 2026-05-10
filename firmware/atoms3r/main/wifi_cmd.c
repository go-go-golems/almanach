/*
 * wifi_cmd.c — Console commands for WiFi management.
 *
 * Registers the following commands with esp_console:
 *   wifi_scan                     — Scan for nearby access points
 *   wifi_connect --ssid <s> --pass <p> — Join a WiFi network
 *   wifi_status                   — Show current connection state and IP
 *   wifi_disconnect               — Disconnect from WiFi
 *   wifi_forget                   — Erase saved WiFi credentials
 */

#include "wifi_cmd.h"

#include <stdio.h>
#include <string.h>
#include "argtable3/argtable3.h"
#include "esp_console.h"
#include "esp_log.h"

#include "nvs_store.h"
#include "provisioning_mgr.h"
#include "wifi_mgr.h"

static const char *TAG __attribute__((unused)) = "wifi_cmd";

/* ========================================================================
 * wifi_scan
 * ======================================================================== */

static int do_wifi_scan(int argc, char **argv)
{
    esp_err_t err = wifi_mgr_scan();
    if (err != ESP_OK) {
        printf("Scan failed: %s\n", esp_err_to_name(err));
        return 1;
    }
    return 0;
}

/* ========================================================================
 * wifi_connect --ssid <ssid> --pass <password>
 * ======================================================================== */

static struct {
    struct arg_str *ssid;
    struct arg_str *pass;
    struct arg_end *end;
} connect_args;

static int do_wifi_connect(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&connect_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, connect_args.end, argv[0]);
        return 1;
    }

    const char *ssid = connect_args.ssid->sval[0];
    const char *pass = connect_args.pass->sval[0];

    printf("Connecting to \"%s\"...\n", ssid);
    esp_err_t err = wifi_mgr_connect(ssid, pass);
    if (err != ESP_OK) {
        if (err == ESP_ERR_TIMEOUT) {
            printf("Connection timed out. Check SSID and password.\n");
        } else {
            printf("Connection failed: %s\n", esp_err_to_name(err));
        }
        return 1;
    }

    /* Save credentials on successful connect */
    char ip[16] = {0};
    wifi_mgr_get_ip(ip, sizeof(ip));
    printf("Connected! IP: %s\n", ip);

    err = nvs_store_save_wifi(ssid, pass);
    if (err != ESP_OK) {
        printf("Warning: failed to save credentials: %s\n", esp_err_to_name(err));
    }
    return 0;
}

/* ========================================================================
 * wifi_status
 * ======================================================================== */

static int do_wifi_status(int argc, char **argv)
{
    if (wifi_mgr_is_connected()) {
        char ip[16] = {0};
        wifi_mgr_get_ip(ip, sizeof(ip));
        printf("WiFi: CONNECTED  IP: %s\n", ip);
    } else {
        printf("WiFi: DISCONNECTED\n");
    }

    /* Show saved SSID */
    char ssid[64] = {0};
    char pass[64] = {0};
    if (nvs_store_load_wifi(ssid, sizeof(ssid), pass, sizeof(pass)) == ESP_OK) {
        printf("Saved SSID: \"%s\"\n", ssid);
    } else {
        printf("No saved credentials\n");
    }
    return 0;
}

/* ========================================================================
 * wifi_disconnect
 * ======================================================================== */

static int do_wifi_disconnect(int argc, char **argv)
{
    wifi_mgr_disconnect();
    printf("WiFi disconnected\n");
    return 0;
}

/* ========================================================================
 * wifi_forget
 * ======================================================================== */

static int do_wifi_forget(int argc, char **argv)
{
    wifi_mgr_disconnect();
    esp_err_t err = nvs_store_erase_wifi();
    if (err != ESP_OK) {
        printf("Error erasing explicit WiFi credentials: %s\n", esp_err_to_name(err));
        return 1;
    }

    err = provisioning_mgr_reset();
    if (err != ESP_OK) {
        printf("Warning: provisioning state reset failed: %s\n", esp_err_to_name(err));
    }

    printf("WiFi credentials and provisioning state erased\n");
    return 0;
}

/* ========================================================================
 * Registration
 * ======================================================================== */

static void reg(const char *name, const char *help,
                esp_console_cmd_func_t func, void *argtable)
{
    const esp_console_cmd_t cmd = {
        .command  = name,
        .help     = help,
        .func     = func,
        .argtable = argtable,
    };
    ESP_ERROR_CHECK(esp_console_cmd_register(&cmd));
}

void wifi_cmd_register(void)
{
    /* wifi_scan — no args */
    reg("wifi_scan", "Scan for nearby WiFi access points",
        do_wifi_scan, NULL);

    /* wifi_connect */
    connect_args.ssid = arg_str1("s", "ssid", "<ssid>", "WiFi SSID");
    connect_args.pass = arg_str1("p", "pass", "<pass>", "WiFi password");
    connect_args.end  = arg_end(2);
    reg("wifi_connect", "Connect to a WiFi network (saves credentials on success)",
        do_wifi_connect, &connect_args);

    /* wifi_status — no args */
    reg("wifi_status", "Show WiFi connection status and saved SSID",
        do_wifi_status, NULL);

    /* wifi_disconnect — no args */
    reg("wifi_disconnect", "Disconnect from WiFi",
        do_wifi_disconnect, NULL);

    /* wifi_forget — no args */
    reg("wifi_forget", "Erase saved WiFi credentials and disconnect",
        do_wifi_forget, NULL);
}
