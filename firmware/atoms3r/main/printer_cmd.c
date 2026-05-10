/*
 * printer_cmd.c — Console commands for the thermal printer.
 *
 * Registers the following commands with esp_console:
 *   printer_init         — Reset printer
 *   printer_text <str>   — Print a text line
 *   printer_feed [n]     — Feed n lines (default 3)
 *   printer_size <n>     — Set font size 0–7
 *   printer_bold <on|off>— Enable/disable bold
 *   printer_align <n>    — Set alignment 0=left 1=center 2=right
 *   printer_barcode <type> <data> — Print barcode
 *   printer_qr <text>    — Print QR code
 *   printer_bitmap_test  — Print test pattern (alternating lines)
 *   printer_baud <rate>  — Change ESP32 UART baud only (recovery)
 *   set_baudrate <rate>  — Tell printer to change baud, then switch ESP32 UART
 *   printer_status       — Read 4-byte printer status
 *   printer_temp         — Read print-head temperature
 *   printer_get_baud     — Read printer-side baud rate
 *   printer_density <n>  — Set density 0..39
 *   printer_speed <n>    — Set mechanism speed
 *   printer_graphics_mode <n> — Set graphics mode 30/31/32
 *   printer_settings_save <baud> <density> <speed> <mode> — Save startup settings to NVS
 *   printer_settings_show  — Show saved startup settings
 *   printer_settings_apply — Apply saved startup settings now
 *   printer_settings_clear — Clear saved startup settings
 */

#include "printer_cmd.h"

#include <stdlib.h>
#include <string.h>
#include "argtable3/argtable3.h"
#include "driver/uart.h"
#include "esp_check.h"
#include "esp_console.h"
#include "esp_log.h"
#include "freertos/FreeRTOS.h"
#include "freertos/task.h"
#include "nvs_flash.h"

#include "nvs_store.h"
#include "printer_drv.h"

/* TAG used for future logging in this module */
#define TAG "printer_cmd"

/* ========================================================================
 * printer_init
 * ======================================================================== */

static int do_printer_init(int argc, char **argv)
{
    esp_err_t err = printer_drv_reset();
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Printer reset OK\n");
    return 0;
}

/* ========================================================================
 * printer_text <string>
 * ======================================================================== */

static struct {
    struct arg_str *text;
    struct arg_end *end;
} text_args;

static int do_printer_text(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&text_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, text_args.end, argv[0]);
        return 1;
    }
    esp_err_t err = printer_drv_print_text(text_args.text->sval[0]);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    return 0;
}

/* ========================================================================
 * printer_feed [n]
 * ======================================================================== */

static struct {
    struct arg_int *lines;
    struct arg_end *end;
} feed_args;

static int do_printer_feed(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&feed_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, feed_args.end, argv[0]);
        return 1;
    }
    int lines = (feed_args.lines->count > 0) ? feed_args.lines->ival[0] : 3;
    if (lines < 1) lines = 1;
    if (lines > 255) lines = 255;
    esp_err_t err = printer_drv_feed((uint8_t)lines);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    return 0;
}

/* ========================================================================
 * printer_size <n>
 * ======================================================================== */

static struct {
    struct arg_int *size;
    struct arg_end *end;
} size_args;

static int do_printer_size(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&size_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, size_args.end, argv[0]);
        return 1;
    }
    int sz = size_args.size->ival[0];
    if (sz < 0) sz = 0;
    if (sz > 7) sz = 7;
    esp_err_t err = printer_drv_set_font_size((uint8_t)sz);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Font size set to %d\n", sz);
    return 0;
}

/* ========================================================================
 * printer_bold <on|off>
 * ======================================================================== */

static struct {
    struct arg_str *state;
    struct arg_end *end;
} bold_args;

static int do_printer_bold(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&bold_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, bold_args.end, argv[0]);
        return 1;
    }
    const char *val = bold_args.state->sval[0];
    bool on;
    if (strcmp(val, "on") == 0) {
        on = true;
    } else if (strcmp(val, "off") == 0) {
        on = false;
    } else {
        printf("Usage: printer_bold <on|off>\n");
        return 1;
    }
    esp_err_t err = printer_drv_set_bold(on);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Bold %s\n", on ? "on" : "off");
    return 0;
}

/* ========================================================================
 * printer_align <n>
 * ======================================================================== */

static struct {
    struct arg_int *align;
    struct arg_end *end;
} align_args;

static int do_printer_align(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&align_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, align_args.end, argv[0]);
        return 1;
    }
    int a = align_args.align->ival[0];
    if (a < 0) a = 0;
    if (a > 2) a = 2;
    esp_err_t err = printer_drv_set_align((uint8_t)a);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Alignment set to %d (%s)\n", a,
           a == 0 ? "left" : a == 1 ? "center" : "right");
    return 0;
}

/* ========================================================================
 * printer_barcode <type> <data>
 * ======================================================================== */

static struct {
    struct arg_str *type;
    struct arg_str *data;
    struct arg_end *end;
} barcode_args;

static int barcode_type_from_str(const char *s, printer_barcode_type_t *out)
{
    if (strcasecmp(s, "UPC_A")    == 0) { *out = PRINTER_BC_UPC_A;   return 0; }
    if (strcasecmp(s, "UPC_E")    == 0) { *out = PRINTER_BC_UPC_E;   return 0; }
    if (strcasecmp(s, "EAN13")    == 0) { *out = PRINTER_BC_EAN13;   return 0; }
    if (strcasecmp(s, "EAN8")     == 0) { *out = PRINTER_BC_EAN8;    return 0; }
    if (strcasecmp(s, "CODE39")   == 0) { *out = PRINTER_BC_CODE39;  return 0; }
    if (strcasecmp(s, "ITF")      == 0) { *out = PRINTER_BC_ITF;     return 0; }
    if (strcasecmp(s, "CODABAR")  == 0) { *out = PRINTER_BC_CODABAR; return 0; }
    if (strcasecmp(s, "CODE93")   == 0) { *out = PRINTER_BC_CODE93;  return 0; }
    if (strcasecmp(s, "CODE128")  == 0) { *out = PRINTER_BC_CODE128; return 0; }
    return -1;
}

static int do_printer_barcode(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&barcode_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, barcode_args.end, argv[0]);
        return 1;
    }

    printer_barcode_type_t type;
    if (barcode_type_from_str(barcode_args.type->sval[0], &type) != 0) {
        printf("Unknown barcode type: %s\n", barcode_args.type->sval[0]);
        printf("Valid types: UPC_A UPC_E EAN13 EAN8 CODE39 ITF CODABAR CODE93 CODE128\n");
        return 1;
    }

    esp_err_t err = printer_drv_print_barcode(type, barcode_args.data->sval[0]);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    return 0;
}

/* ========================================================================
 * printer_qr <text>
 * ======================================================================== */

static struct {
    struct arg_str *data;
    struct arg_end *end;
} qr_args;

static int do_printer_qr(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&qr_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, qr_args.end, argv[0]);
        return 1;
    }
    esp_err_t err = printer_drv_print_qr(PRINTER_QR_EC_M, qr_args.data->sval[0]);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    return 0;
}

/* ========================================================================
 * printer_bitmap_test — print alternating-line test pattern
 * ======================================================================== */

static int do_printer_bitmap_test(int argc, char **argv)
{
    const int width_dots    = 384;
    const int bytes_per_row = width_dots / 8; /* 48 */
    const int height        = 100;

    printf("Printing test bitmap: %d x %d pixels...\n", width_dots, height);

    /* Build the bitmap in a buffer.  48 * 100 = 4800 bytes — fits in
     * internal SRAM easily, but we use PSRAM via heap_caps for
     * consistency with larger future images. */
    size_t total = (size_t)bytes_per_row * height;
    uint8_t *bitmap = (uint8_t *)malloc(total);
    if (!bitmap) {
        printf("Failed to allocate %zu bytes for bitmap\n", total);
        return 1;
    }

    for (int row = 0; row < height; row++) {
        if (row % 2 == 0) {
            memset(bitmap + row * bytes_per_row, 0xFF, bytes_per_row);
        } else {
            memset(bitmap + row * bytes_per_row, 0x00, bytes_per_row);
        }
    }

    esp_err_t err = printer_drv_print_bitmap(width_dots, height, bitmap);
    free(bitmap);

    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Bitmap sent\n");
    return 0;
}

/* ========================================================================
 * printer_probe — query printer status and dump RX buffer
 * ======================================================================== */

static int do_printer_probe(int argc, char **argv)
{
    printf("=== Printer probe ===\n");
    printf("UART config: port=UART%d TX=GPIO%d RX=GPIO%d CTS=GPIO%d baud=%d%s\n",
           PRINTER_UART_NUM, PRINTER_TX_GPIO, PRINTER_RX_GPIO, PRINTER_CTS_GPIO,
           printer_drv_get_baud(),
           printer_drv_is_swapped() ? " (SWAPPED)" : "");

    /* 1. Drain any stale RX bytes */
    printf("\nDraining RX buffer...\n");
    int drained = printer_drv_drain_rx();
    printf("Drained %d stale bytes\n", drained);

    /* 2. Query real-time status: DLE EOT n for n=1..4 */
    printf("\nQuerying printer status (4 DLE EOT queries)...\n");
    bool any_response = false;
    for (uint8_t n = 1; n <= 4; n++) {
        uint8_t resp = 0;
        esp_err_t err = printer_drv_query_status(n, &resp);
        if (err == ESP_OK) {
            any_response = true;
            const char *name[] = { "", "printer", "offline", "error", "paper" };
            printf("  Status n=%d (%s): 0x%02X\n", n, name[n], resp);
        } else {
            printf("  Status n=%d: NO RESPONSE\n", n);
        }
    }

    /* 3. Send ESC @ (init) and check for response */
    printf("\nSending ESC @ (init)...\n");
    esp_err_t reset_err = printer_drv_reset();
    printf("  send result: %s\n", esp_err_to_name(reset_err));

    /* 4. Wait and drain again */
    vTaskDelay(pdMS_TO_TICKS(200));
    int after = printer_drv_drain_rx();
    if (after > 0) {
        printf("  Got %d bytes after init (printer is alive!)\n", after);
        any_response = true;
    }

    printf("\n=== Result: %s ===\n",
           any_response ? "PRINTER RESPONDED" : "NO RESPONSE (check wiring/power)");

    if (!any_response) {
        printf("\nTroubleshooting:\n");
        printf("  1. Is 12V power supply connected to the printer carrier board?\n");
        printf("  2. Is the HY2.0-4P cable plugged in?\n");
        printf("  3. Check TX<->RX crossover: ESP TX(GPIO%d) -> printer RX\n", PRINTER_TX_GPIO);
        printf("  4. GND must be shared between AtomS3R and printer board\n");
        printf("  5. Are GPIO%d/GPIO%d/GPIO%d the correct pins for your K118 cable?\n",
               PRINTER_TX_GPIO, PRINTER_RX_GPIO, PRINTER_CTS_GPIO);
        printf("     (K118 designed for ATOM Lite: TX=GPIO23 RX=GPIO33 CTS=GPIO19)\n");
        printf("  6. Try 'printer_swap on' to flip TX<->RX and re-probe\n");
    }

    return any_response ? 0 : 1;
}

/* ========================================================================
 * printer_raw <hex> — send raw hex bytes
 * ======================================================================== */

static struct {
    struct arg_str *hex;
    struct arg_end *end;
} raw_args;

static int do_printer_raw(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&raw_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, raw_args.end, argv[0]);
        return 1;
    }

    /* Parse hex string like "1B40" or "1B 40" */
    const char *hex = raw_args.hex->sval[0];
    size_t hex_len = strlen(hex);
    uint8_t buf[256];
    size_t buf_len = 0;

    /* Remove spaces */
    char clean[256];
    size_t ci = 0;
    for (size_t i = 0; i < hex_len && ci < sizeof(clean) - 1; i++) {
        if (hex[i] != ' ') clean[ci++] = hex[i];
    }
    clean[ci] = '\0';

    if (ci % 2 != 0) {
        printf("Hex string must have even length (got %zu)\n", ci);
        return 1;
    }

    for (size_t i = 0; i < ci; i += 2) {
        unsigned int val;
        if (sscanf(clean + i, "%2x", &val) != 1) {
            printf("Invalid hex at position %zu\n", i);
            return 1;
        }
        buf[buf_len++] = (uint8_t)val;
    }

    printf("Sending %zu raw bytes\n", buf_len);
    esp_err_t err = printer_drv_send_raw(buf, buf_len);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }

    /* Wait and check for response */
    vTaskDelay(pdMS_TO_TICKS(200));
    int got = printer_drv_drain_rx();
    if (got > 0) printf("(printer sent %d bytes back)\n", got);
    return 0;
}

/* ========================================================================
 * printer_swap <on|off> — toggle TX<->RX crossover
 * ======================================================================== */

static struct {
    struct arg_str *state;
    struct arg_end *end;
} swap_args;

static int do_printer_swap(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&swap_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, swap_args.end, argv[0]);
        return 1;
    }
    const char *val = swap_args.state->sval[0];
    bool swap;
    if (strcmp(val, "on") == 0) {
        swap = true;
    } else if (strcmp(val, "off") == 0) {
        swap = false;
    } else {
        printf("Usage: printer_swap <on|off>\n");
        return 1;
    }
    esp_err_t err = printer_drv_swap_pins(swap);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Pins %s: TX=GPIO%d RX=GPIO%d\n",
           swap ? "SWAPPED" : "NORMAL",
           swap ? PRINTER_RX_GPIO : PRINTER_TX_GPIO,
           swap ? PRINTER_TX_GPIO : PRINTER_RX_GPIO);
    return 0;
}

/* ========================================================================
 * printer_baud <rate> — change ESP32 UART baud rate only (recovery)
 * ======================================================================== */

static struct {
    struct arg_int *rate;
    struct arg_end *end;
} baud_args;

static const char *supported_baud_list(void)
{
    return "9600, 19200, 38400, 57600, 115200, 230400, 460800, 921600";
}

static bool is_supported_baud(int rate)
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

static bool is_experimental_baud(int rate)
{
    return rate > 115200;
}

static int do_printer_baud(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&baud_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, baud_args.end, argv[0]);
        return 1;
    }
    int rate = baud_args.rate->ival[0];
    if (!is_supported_baud(rate)) {
        printf("Unsupported baud rate: %d\n", rate);
        printf("Supported: %s\n", supported_baud_list());
        return 1;
    }
    if (is_experimental_baud(rate)) {
        printf("Warning: rates above 115200 are experimental on K118.\n");
    }
    esp_err_t err = printer_drv_set_baud(rate);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("ESP32 UART baud rate set to %d (printer was NOT commanded)\n", rate);
    return 0;
}

/* ========================================================================
 * set_baudrate <rate> — command printer baud, then switch ESP32 UART
 * ======================================================================== */

static struct {
    struct arg_int *rate;
    struct arg_end *end;
} set_baudrate_args;

static int do_set_baudrate(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&set_baudrate_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, set_baudrate_args.end, argv[0]);
        return 1;
    }
    int rate = set_baudrate_args.rate->ival[0];
    if (!is_supported_baud(rate)) {
        printf("Unsupported baud rate: %d\n", rate);
        printf("Supported: %s\n", supported_baud_list());
        return 1;
    }

    printf("Sending K118 Set Baud Rate command for %d baud...\n", rate);
    if (is_experimental_baud(rate)) {
        printf("Warning: rates above 115200 are experimental on K118; be ready to power-cycle.\n");
    }
    printf("If communication is lost, use printer_baud <rate> to resync the ESP32 side,\n");
    printf("or power-cycle the printer to return it to its default 9600 baud.\n");

    esp_err_t err = printer_drv_set_printer_baudrate(rate);
    if (err != ESP_OK) {
        printf("Error: %s\n", esp_err_to_name(err));
        return 1;
    }
    printf("Printer and ESP32 UART baud rate set to %d\n", rate);
    return 0;
}

/* ========================================================================
 * printer_status / printer_temp / printer_get_baud / tuning commands
 * ======================================================================== */

static int do_printer_status(int argc, char **argv)
{
    (void)argc; (void)argv;
    printer_status_t st;
    esp_err_t err = printer_drv_query_status4(&st);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("raw: %02X %02X %02X %02X\n", st.raw[0], st.raw[1], st.raw[2], st.raw[3]);
    printf("buffer_full=%s cover_open=%s feed_key=%s cutter_error=%s auto_error=%s overheated=%s paper_near_end=%s paper_out=%s\n",
           st.buffer_full ? "yes" : "no",
           st.cover_open ? "yes" : "no",
           st.feed_key_active ? "yes" : "no",
           st.cutter_error ? "yes" : "no",
           st.auto_recoverable_error ? "yes" : "no",
           st.overheated ? "yes" : "no",
           st.paper_near_end ? "yes" : "no",
           st.paper_out ? "yes" : "no");
    return 0;
}

static int do_printer_temp(int argc, char **argv)
{
    (void)argc; (void)argv;
    char raw[64];
    int temp = -1;
    esp_err_t err = printer_drv_query_temperature(&temp, raw, sizeof(raw));
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("temperature_c=%d raw=%s\n", temp, raw);
    return 0;
}

static int do_printer_get_baud(int argc, char **argv)
{
    (void)argc; (void)argv;
    char raw[80];
    int printer_baud = -1;
    esp_err_t err = printer_drv_query_printer_baud(&printer_baud, raw, sizeof(raw));
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("esp32_baud=%d printer_baud=%d raw=%s\n", printer_drv_get_baud(), printer_baud, raw);
    return 0;
}

static struct { struct arg_int *value; struct arg_end *end; } density_args;
static int do_printer_density(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&density_args);
    if (nerrors != 0) { arg_print_errors(stderr, density_args.end, argv[0]); return 1; }
    int v = density_args.value->ival[0];
    if (v < 0 || v > 39) { printf("Density must be 0..39\n"); return 1; }
    esp_err_t err = printer_drv_set_density((uint8_t)v);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("Density set to %d\n", v);
    return 0;
}

static struct { struct arg_int *value; struct arg_end *end; } speed_args;
static bool is_supported_speed(int v)
{
    static const int speeds[] = { 25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220 };
    for (size_t i = 0; i < sizeof(speeds) / sizeof(speeds[0]); i++) if (speeds[i] == v) return true;
    return false;
}
static int do_printer_speed(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&speed_args);
    if (nerrors != 0) { arg_print_errors(stderr, speed_args.end, argv[0]); return 1; }
    int v = speed_args.value->ival[0];
    if (!is_supported_speed(v)) { printf("Speed must be one of: 25,30,37,50,56,62,70,80,90,100,120,150,180,200,220\n"); return 1; }
    esp_err_t err = printer_drv_set_speed((uint8_t)v);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("Speed set to %d mm/s\n", v);
    return 0;
}

static struct { struct arg_int *value; struct arg_end *end; } graphics_mode_args;
static int do_printer_graphics_mode(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&graphics_mode_args);
    if (nerrors != 0) { arg_print_errors(stderr, graphics_mode_args.end, argv[0]); return 1; }
    int v = graphics_mode_args.value->ival[0];
    if (v != 30 && v != 31 && v != 32) { printf("Graphics mode must be 30=BLE, 31=adaptive, or 32=constant\n"); return 1; }
    esp_err_t err = printer_drv_set_graphics_mode((uint8_t)v);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("Graphics mode set to %d (%s)\n", v, v == 30 ? "BLE" : (v == 31 ? "adaptive" : "constant"));
    return 0;
}

/* ========================================================================
 * Saved printer startup settings
 * ======================================================================== */

static struct {
    struct arg_int *baud;
    struct arg_int *density;
    struct arg_int *speed;
    struct arg_int *mode;
    struct arg_end *end;
} printer_settings_save_args;

static bool validate_printer_settings(const printer_settings_t *settings)
{
    return settings != NULL &&
           is_supported_baud(settings->baud) &&
           settings->density >= 0 && settings->density <= 39 &&
           is_supported_speed(settings->speed) &&
           (settings->graphics_mode == 30 || settings->graphics_mode == 31 || settings->graphics_mode == 32);
}

static void print_printer_settings(const char *prefix, const printer_settings_t *settings)
{
    printf("%s baud=%ld density=%ld speed=%ld graphics_mode=%ld (%s)\n",
           prefix,
           (long)settings->baud,
           (long)settings->density,
           (long)settings->speed,
           (long)settings->graphics_mode,
           settings->graphics_mode == 30 ? "BLE" : (settings->graphics_mode == 31 ? "adaptive" : "constant"));
}

static esp_err_t apply_printer_settings(const printer_settings_t *settings)
{
    if (!validate_printer_settings(settings)) {
        return ESP_ERR_INVALID_ARG;
    }

    /* Startup/saved baud is applied to the ESP32 UART side directly. This is the
     * right behavior when the K118 has already persisted its own baud setting.
     * If the printer has been power-cycled back to 9600, recover with
     * printer_baud 9600 or clear the saved settings. */
    ESP_RETURN_ON_ERROR(printer_drv_set_baud(settings->baud), TAG, "set saved UART baud");
    ESP_RETURN_ON_ERROR(printer_drv_set_density((uint8_t)settings->density), TAG, "set saved density");
    ESP_RETURN_ON_ERROR(printer_drv_set_speed((uint8_t)settings->speed), TAG, "set saved speed");
    ESP_RETURN_ON_ERROR(printer_drv_set_graphics_mode((uint8_t)settings->graphics_mode), TAG, "set saved graphics mode");
    return ESP_OK;
}

static int do_printer_settings_save(int argc, char **argv)
{
    int nerrors = arg_parse(argc, argv, (void **)&printer_settings_save_args);
    if (nerrors != 0) {
        arg_print_errors(stderr, printer_settings_save_args.end, argv[0]);
        return 1;
    }

    printer_settings_t settings = {
        .baud = printer_settings_save_args.baud->ival[0],
        .density = printer_settings_save_args.density->ival[0],
        .speed = printer_settings_save_args.speed->ival[0],
        .graphics_mode = printer_settings_save_args.mode->ival[0],
    };

    if (!validate_printer_settings(&settings)) {
        printf("Invalid settings. Baud must be one of %s; density 0..39; speed one of 25,30,37,50,56,62,70,80,90,100,120,150,180,200,220; mode 30/31/32.\n",
               supported_baud_list());
        return 1;
    }

    esp_err_t err = nvs_store_save_printer_settings(&settings);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    print_printer_settings("Saved printer startup settings:", &settings);
    printf("These settings are applied on boot. Saved baud sets the ESP32 UART side directly; use set_baudrate once first so the K118 is also at that baud.\n");
    return 0;
}

static int do_printer_settings_show(int argc, char **argv)
{
    (void)argc; (void)argv;
    printer_settings_t settings;
    esp_err_t err = nvs_store_load_printer_settings(&settings);
    if (err == ESP_ERR_NVS_NOT_FOUND) {
        printf("No saved printer startup settings.\n");
        return 0;
    }
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    print_printer_settings("Saved printer startup settings:", &settings);
    return 0;
}

static int do_printer_settings_apply(int argc, char **argv)
{
    (void)argc; (void)argv;
    printer_settings_t settings;
    esp_err_t err = nvs_store_load_printer_settings(&settings);
    if (err == ESP_ERR_NVS_NOT_FOUND) {
        printf("No saved printer startup settings.\n");
        return 1;
    }
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    err = apply_printer_settings(&settings);
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    print_printer_settings("Applied printer startup settings:", &settings);
    return 0;
}

static int do_printer_settings_clear(int argc, char **argv)
{
    (void)argc; (void)argv;
    esp_err_t err = nvs_store_erase_printer_settings();
    if (err != ESP_OK) { printf("Error: %s\n", esp_err_to_name(err)); return 1; }
    printf("Saved printer startup settings cleared. Defaults will be used on next boot.\n");
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

void printer_cmd_register(void)
{
    /* ---- printer_init (no args) ---- */
    reg("printer_init", "Reset the thermal printer (ESC @)",
        do_printer_init, NULL);

    /* ---- printer_text ---- */
    text_args.text = arg_str1(NULL, NULL, "<text>", "Text to print");
    text_args.end  = arg_end(1);
    reg("printer_text", "Print a line of text", do_printer_text, &text_args);

    /* ---- printer_feed ---- */
    feed_args.lines = arg_int0(NULL, NULL, "[<n>]", "Number of lines to feed (default 3)");
    feed_args.end   = arg_end(1);
    reg("printer_feed", "Feed paper by n lines", do_printer_feed, &feed_args);

    /* ---- printer_size ---- */
    size_args.size = arg_int1(NULL, NULL, "<n>", "Font size 0-7");
    size_args.end  = arg_end(1);
    reg("printer_size", "Set font size (0-7)", do_printer_size, &size_args);

    /* ---- printer_bold ---- */
    bold_args.state = arg_str1(NULL, NULL, "<on|off>", "Enable or disable bold");
    bold_args.end   = arg_end(1);
    reg("printer_bold", "Enable/disable bold text", do_printer_bold, &bold_args);

    /* ---- printer_align ---- */
    align_args.align = arg_int1(NULL, NULL, "<n>", "Alignment: 0=left 1=center 2=right");
    align_args.end   = arg_end(1);
    reg("printer_align", "Set text alignment", do_printer_align, &align_args);

    /* ---- printer_barcode ---- */
    barcode_args.type = arg_str1(NULL, NULL, "<type>", "Barcode type (CODE128, EAN13, ...)");
    barcode_args.data = arg_str1(NULL, NULL, "<data>", "Barcode data string");
    barcode_args.end  = arg_end(2);
    reg("printer_barcode", "Print a barcode", do_printer_barcode, &barcode_args);

    /* ---- printer_qr ---- */
    qr_args.data = arg_str1(NULL, NULL, "<text>", "QR code content");
    qr_args.end  = arg_end(1);
    reg("printer_qr", "Print a QR code", do_printer_qr, &qr_args);

    /* ---- printer_bitmap_test (no args) ---- */
    reg("printer_bitmap_test", "Print a test bitmap pattern (alternating lines)",
        do_printer_bitmap_test, NULL);

    /* ---- printer_probe (no args) ---- */
    reg("printer_probe", "Query printer status and diagnose connection",
        do_printer_probe, NULL);

    /* ---- printer_raw <hex> ---- */
    raw_args.hex = arg_str1(NULL, NULL, "<hex>", "Hex bytes to send, e.g. 1B40 or \"1B 40\"");
    raw_args.end = arg_end(1);
    reg("printer_raw", "Send raw hex bytes to the printer",
        do_printer_raw, &raw_args);

    /* ---- printer_swap <on|off> ---- */
    swap_args.state = arg_str1(NULL, NULL, "<on|off>", "Swap TX<->RX pins");
    swap_args.end   = arg_end(1);
    reg("printer_swap", "Swap TX/RX pins (try if printer_probe gets no response)",
        do_printer_swap, &swap_args);

    /* ---- printer_baud <rate> ---- */
    baud_args.rate = arg_int1(NULL, NULL, "<rate>", "ESP32 UART baud only: 9600..921600");
    baud_args.end  = arg_end(1);
    reg("printer_baud", "Change ESP32 UART baud only (recovery)",
        do_printer_baud, &baud_args);

    /* ---- set_baudrate <rate> ---- */
    set_baudrate_args.rate = arg_int1(NULL, NULL, "<rate>", "Printer+ESP32 baud: 9600..921600 (above 115200 experimental)");
    set_baudrate_args.end  = arg_end(1);
    reg("set_baudrate", "Tell K118 to change baud, then switch ESP32 UART",
        do_set_baudrate, &set_baudrate_args);

    reg("printer_status", "Read 4-byte printer status (GS a n)",
        do_printer_status, NULL);
    reg("printer_temp", "Read printer temperature (GS g 6)",
        do_printer_temp, NULL);
    reg("printer_get_baud", "Read printer-side baud rate (GS g 7)",
        do_printer_get_baud, NULL);

    density_args.value = arg_int1(NULL, NULL, "<0-39>", "Print density, 39 darkest");
    density_args.end = arg_end(1);
    reg("printer_density", "Set print density (ESC ## STDP)",
        do_printer_density, &density_args);

    speed_args.value = arg_int1(NULL, NULL, "<speed>", "25/30/37/50/56/62/70/80/90/100/120/150/180/200/220");
    speed_args.end = arg_end(1);
    reg("printer_speed", "Set print speed (ESC ## STSP)",
        do_printer_speed, &speed_args);

    graphics_mode_args.value = arg_int1(NULL, NULL, "<30|31|32>", "30=BLE, 31=adaptive, 32=constant");
    graphics_mode_args.end = arg_end(1);
    reg("printer_graphics_mode", "Set graphics print mode (ESC ## SPSM)",
        do_printer_graphics_mode, &graphics_mode_args);

    printer_settings_save_args.baud = arg_int1(NULL, NULL, "<baud>", "Startup baud: 9600..921600");
    printer_settings_save_args.density = arg_int1(NULL, NULL, "<0-39>", "Startup print density");
    printer_settings_save_args.speed = arg_int1(NULL, NULL, "<speed>", "Startup speed: 25/30/37/50/56/62/70/80/90/100/120/150/180/200/220");
    printer_settings_save_args.mode = arg_int1(NULL, NULL, "<30|31|32>", "Startup graphics mode: 30=BLE, 31=adaptive, 32=constant");
    printer_settings_save_args.end = arg_end(4);
    reg("printer_settings_save", "Save printer startup settings to NVS",
        do_printer_settings_save, &printer_settings_save_args);
    reg("printer_settings_show", "Show saved printer startup settings",
        do_printer_settings_show, NULL);
    reg("printer_settings_apply", "Apply saved printer startup settings now",
        do_printer_settings_apply, NULL);
    reg("printer_settings_clear", "Clear saved printer startup settings",
        do_printer_settings_clear, NULL);
}
