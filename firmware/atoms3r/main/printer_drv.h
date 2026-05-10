#pragma once

#include <stdbool.h>
#include <stddef.h>
#include <stdint.h>
#include "esp_err.h"

/* Printer UART configuration for AtomS3R Lite + K118
 *
 * The K118 was designed for the ATOM Lite (TX=GPIO23, RX=GPIO33, CTS=GPIO19).
 * On the AtomS3R Lite, those same header positions map to:
 *   TX  = GPIO8  (ESP32 sends TO printer)
 *   RX  = GPIO7  (ESP32 reads FROM printer)
 *   CTS = GPIO6  (clear-to-send, optional)
 */
#define PRINTER_UART_NUM   UART_NUM_1
#define PRINTER_TX_GPIO    8
#define PRINTER_RX_GPIO    7
#define PRINTER_CTS_GPIO   6
#define PRINTER_BAUD       9600

/* Barcode types (ESC/POS) */
typedef enum {
    PRINTER_BC_UPC_A    = 0x41,
    PRINTER_BC_UPC_E    = 0x42,
    PRINTER_BC_EAN13    = 0x43,
    PRINTER_BC_EAN8     = 0x44,
    PRINTER_BC_CODE39   = 0x45,
    PRINTER_BC_ITF      = 0x46,
    PRINTER_BC_CODABAR  = 0x47,
    PRINTER_BC_CODE93   = 0x48,
    PRINTER_BC_CODE128  = 0x49,
} printer_barcode_type_t;

/* QR error correction levels */
typedef enum {
    PRINTER_QR_EC_L = 0x48,  /* Low    (~7%)  */
    PRINTER_QR_EC_M = 0x49,  /* Medium (~15%) */
    PRINTER_QR_EC_Q = 0x4A,  /* Quartile (~25%) */
    PRINTER_QR_EC_H = 0x4B,  /* High   (~30%) */
} printer_qr_ec_level_t;

typedef struct {
    uint8_t raw[4];
    bool buffer_full;
    bool cover_open;
    bool feed_key_active;
    bool cutter_error;
    bool auto_recoverable_error;
    bool overheated;
    bool paper_near_end;
    bool paper_out;
} printer_status_t;

/**
 * Initialize the printer UART (9600 8N1 on the K118 header pins) and send ESC @.
 */
esp_err_t printer_drv_init(void);

/**
 * Send ESC @ to reset the printer to power-on defaults.
 */
esp_err_t printer_drv_reset(void);

/**
 * Print a plain-text string (ASCII) followed by a line feed.
 */
esp_err_t printer_drv_print_text(const char *text);

/**
 * Feed paper by `lines` lines.
 */
esp_err_t printer_drv_feed(uint8_t lines);

/**
 * Set font size. Both width and height are set to `size` (0–7).
 * 0 = normal, 7 = 8x magnification in each dimension.
 */
esp_err_t printer_drv_set_font_size(uint8_t size);

/**
 * Enable or disable bold (emphasized) printing.
 */
esp_err_t printer_drv_set_bold(bool on);

/**
 * Set text alignment: 0=left, 1=center, 2=right.
 */
esp_err_t printer_drv_set_align(uint8_t align);

/**
 * Print a barcode of the given type containing `data`.
 */
esp_err_t printer_drv_print_barcode(printer_barcode_type_t type,
                                     const char *data);

/**
 * Print a QR code containing `data` with the given error correction level.
 */
esp_err_t printer_drv_print_qr(printer_qr_ec_level_t ec_level,
                                 const char *data);

/**
 * Send *only* the GS v 0 header for a bitmap (no pixel data).
 * Used for streaming: call this first, then send raw pixel bytes.
 */
esp_err_t printer_drv_print_bitmap_header(uint16_t width, uint16_t height);

/**
 * Print a 1-bit monochrome raster bitmap (MSB first).
 * `width` must be a multiple of 8. `pixels` points to (width/8)*height bytes.
 * Sends one complete raster command. UART CTS flow control is enabled so the
 * printer can pause TX without losing bytes when it is busy.
 */
esp_err_t printer_drv_print_bitmap(uint16_t width, uint16_t height,
                                    const uint8_t *pixels);

/**
 * Diagnostic fallback: print a bitmap as multiple complete GS v 0 raster bands.
 *
 * This is useful only when CTS is unavailable. It may create visible regular
 * seams at band boundaries, so the normal bitmap path does not use it.
 */
esp_err_t printer_drv_print_bitmap_banded(uint16_t width, uint16_t height,
                                           const uint8_t *pixels,
                                           uint16_t band_rows,
                                           uint16_t delay_ms);

/**
 * Query printer real-time status using DLE EOT n.
 * `n`: 1=printer status, 2=offline status, 3=error status, 4=paper sensor.
 * Writes the response byte into `*out`. Returns ESP_OK if a byte was received.
 */
esp_err_t printer_drv_query_status(uint8_t n, uint8_t *out);

/**
 * Query the 4-byte K118 status packet using GS a n.
 */
esp_err_t printer_drv_query_status4(printer_status_t *out);

/**
 * Query printer temperature via GS g 6. Returns parsed Celsius value when
 * possible and copies raw response text into out_raw.
 */
esp_err_t printer_drv_query_temperature(int *out_celsius, char *out_raw, size_t out_raw_len);

/**
 * Query printer-side baud rate via GS g 7. Returns parsed baud when possible
 * and copies raw response text into out_raw.
 */
esp_err_t printer_drv_query_printer_baud(int *out_baud, char *out_raw, size_t out_raw_len);

/** Set K118 print density, 0..39 (39 darkest). */
esp_err_t printer_drv_set_density(uint8_t density);

/** Set K118 mechanism speed. Value must be one of the manual's speed table. */
esp_err_t printer_drv_set_speed(uint8_t speed);

/** Set K118 graphics print mode: 30=BLE, 31=adaptive, 32=constant. */
esp_err_t printer_drv_set_graphics_mode(uint8_t mode);

/**
 * Read any pending bytes from the printer RX buffer and log them.
 * Returns the number of bytes read.
 */
int printer_drv_drain_rx(void);

/**
 * Swap TX and RX pins (in case the cable is straight-through instead of
 * crossed).  Call `printer_swap on` to swap, `printer_swap off` to restore.
 * Takes effect immediately on the live UART — no re-init needed.
 */
esp_err_t printer_drv_swap_pins(bool swap);

/**
 * Returns true if pins are currently swapped.
 */
bool printer_drv_is_swapped(void);

/**
 * Enable or disable CTS hardware flow control at runtime.
 * Disabling flow control also detaches the CTS pin, matching the older simple
 * Atom Lite provisioning example for printer diagnostics.
 */
esp_err_t printer_drv_set_flow_control(bool enabled);

/**
 * Returns true when CTS hardware flow control is currently enabled.
 */
bool printer_drv_is_flow_control_enabled(void);

/**
 * Change only the ESP32 UART baud rate at runtime. Default is 9600.
 * Use this as a recovery/diagnostic command when the host and printer are
 * already out of sync.
 */
esp_err_t printer_drv_set_baud(int baud);

/**
 * Send the K118 Set Baud Rate command to the printer, wait for it to leave
 * the current UART baud, then switch the ESP32 UART to the same rate.
 *
 * M5Stack documents 9600 and 115200 examples. Higher common UART rates are
 * exposed experimentally by encoding the requested baud as the same little-
 * endian uint32 payload.
 *
 * Command format from M5Stack docs:
 *   1B 23 23 53 42 44 52 <baud little-endian uint32>
 */
esp_err_t printer_drv_set_printer_baudrate(int baud);

/**
 * Returns the current ESP32 UART baud rate.
 */
int printer_drv_get_baud(void);

/**
 * Send raw bytes directly (for probing / testing).
 * Waits for TX complete.
 */
esp_err_t printer_drv_send_raw(const uint8_t *data, size_t len);

/**
 * Write bytes to UART without waiting for TX complete.
 * Use for bitmap streaming where you want back-to-back chunks.
 */
esp_err_t printer_drv_write_no_wait(const uint8_t *data, size_t len);
