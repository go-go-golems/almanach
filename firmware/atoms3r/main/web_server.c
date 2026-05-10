/*
 * web_server.c — Minimal HTTP server for SToMS3R.
 *
 * Serves the embedded index.html at / and provides:
 *   POST /api/print/text    — { "text": "..." }
 *   POST /api/print/bitmap  — raw 1-bit bitmap body (width x height in headers)
 *   GET  /api/status        — JSON: wifi, printer state
 *   GET  /api/printer/status, /temp, /baud
 *   POST /api/printer/density, /speed, /graphics-mode
 */

#include "web_server.h"

#include <stdlib.h>
#include <string.h>
#include "driver/uart.h"
#include "esp_http_server.h"
#include "esp_log.h"

#include "printer_drv.h"
#include "wifi_mgr.h"

static const char *TAG = "web_server";
static httpd_handle_t s_server = NULL;

/* ---- Embedded HTML --------------------------------------------------- */

extern const uint8_t index_html_start[] asm("_binary_index_html_start");
extern const uint8_t index_html_end[]   asm("_binary_index_html_end");

/* Almanach Studio embedded assets (precompiled React SPA). */
extern const uint8_t almanach_index_html_start[] asm("_binary_almanach_html_start");
extern const uint8_t almanach_index_html_end[]   asm("_binary_almanach_html_end");
extern const uint8_t almanach_bundle_js_start[] asm("_binary_almanach_bundle_js_start");
extern const uint8_t almanach_bundle_js_end[]   asm("_binary_almanach_bundle_js_end");

/* ---- Helper: read full body ------------------------------------------ */

static char *read_body(httpd_req_t *req, size_t *out_len)
{
    size_t len = (size_t)req->content_len;
    if (len == 0 || len > 256 * 1024) return NULL;

    char *buf = malloc(len + 1);
    if (!buf) return NULL;

    size_t off = 0;
    while (off < len) {
        int n = httpd_req_recv(req, buf + off, len - off);
        if (n <= 0) { free(buf); return NULL; }
        off += (size_t)n;
    }
    buf[off] = '\0';
    if (out_len) *out_len = off;
    return buf;
}

/* ---- Helper: send JSON / parse tiny JSON ------------------------------ */

static bool json_get_int(const char *body, const char *key_name, int *out)
{
    if (!body || !key_name || !out) return false;
    char key[48];
    snprintf(key, sizeof(key), "\"%s\"", key_name);
    char *p = strstr(body, key);
    if (!p) return false;
    p = strchr(p + strlen(key), ':');
    if (!p) return false;
    p++;
    while (*p == ' ' || *p == '\t') p++;
    *out = atoi(p);
    return true;
}

static void send_json_ok(httpd_req_t *req)
{
    httpd_resp_set_type(req, "application/json");
    httpd_resp_sendstr(req, "{\"ok\":true}");
}

static void send_json_error(httpd_req_t *req, const char *msg)
{
    httpd_resp_set_type(req, "application/json");
    char buf[128];
    snprintf(buf, sizeof(buf), "{\"ok\":false,\"error\":\"%s\"}", msg);
    httpd_resp_sendstr(req, buf);
}

/* ---- GET / — serve index.html ---------------------------------------- */

static esp_err_t root_get(httpd_req_t *req)
{
    httpd_resp_set_type(req, "text/html; charset=utf-8");
    httpd_resp_set_hdr(req, "Cache-Control", "no-store");
    size_t len = (size_t)(index_html_end - index_html_start);
    return httpd_resp_send(req, (const char *)index_html_start, len);
}

/* ---- GET /api/status ------------------------------------------------- */

static esp_err_t api_status_get(httpd_req_t *req)
{
    char ip[16] = "disconnected";
    if (wifi_mgr_is_connected()) {
        wifi_mgr_get_ip(ip, sizeof(ip));
    }

    char buf[256];
    snprintf(buf, sizeof(buf),
             "{\"ok\":true,\"wifi\":{\"connected\":%s,\"ip\":\"%s\"},"
             "\"printer\":{\"baud\":%d,\"swapped\":%s}}",
             wifi_mgr_is_connected() ? "true" : "false",
             ip,
             printer_drv_get_baud(),
             printer_drv_is_swapped() ? "true" : "false");

    httpd_resp_set_type(req, "application/json");
    return httpd_resp_sendstr(req, buf);
}

/* ---- POST /api/print/text -------------------------------------------- */

static esp_err_t api_print_text_post(httpd_req_t *req)
{
    size_t body_len = 0;
    char *body = read_body(req, &body_len);
    if (!body) {
        send_json_error(req, "empty or too large body");
        return ESP_FAIL;
    }

    /* Find "text":"..." in JSON */
    char *key = strstr(body, "\"text\"");
    if (!key) {
        free(body);
        send_json_error(req, "missing \"text\" field");
        return ESP_FAIL;
    }
    /* key points to "text":"value"} — skip past key to the colon */
    char *colon = strchr(key + 6, ':');
    if (!colon) { free(body); send_json_error(req, "bad JSON: no colon"); return ESP_FAIL; }
    /* Find opening quote of the value */
    char *text_start = strchr(colon, '"');
    if (!text_start) { free(body); send_json_error(req, "bad JSON: no value"); return ESP_FAIL; }
    text_start++; /* skip the opening quote */
    char *text_end = strchr(text_start, '"');
    if (!text_end) { free(body); send_json_error(req, "bad JSON: no closing quote"); return ESP_FAIL; }

    /* Null-terminate the text */
    *text_end = '\0';

    esp_err_t err = printer_drv_print_text(text_start);
    free(body);

    if (err != ESP_OK) {
        send_json_error(req, esp_err_to_name(err));
        return ESP_FAIL;
    }

    send_json_ok(req);
    return ESP_OK;
}

/* ---- Printer diagnostics / settings API ------------------------------- */

static esp_err_t api_printer_status_get(httpd_req_t *req)
{
    printer_status_t st;
    esp_err_t err = printer_drv_query_status4(&st);
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }

    char buf[512];
    snprintf(buf, sizeof(buf),
             "{\"ok\":true,\"raw\":[%u,%u,%u,%u],"
             "\"buffer_full\":%s,\"cover_open\":%s,\"feed_key_active\":%s,"
             "\"cutter_error\":%s,\"auto_recoverable_error\":%s,"
             "\"overheated\":%s,\"paper_near_end\":%s,\"paper_out\":%s}",
             st.raw[0], st.raw[1], st.raw[2], st.raw[3],
             st.buffer_full ? "true" : "false",
             st.cover_open ? "true" : "false",
             st.feed_key_active ? "true" : "false",
             st.cutter_error ? "true" : "false",
             st.auto_recoverable_error ? "true" : "false",
             st.overheated ? "true" : "false",
             st.paper_near_end ? "true" : "false",
             st.paper_out ? "true" : "false");
    httpd_resp_set_type(req, "application/json");
    return httpd_resp_sendstr(req, buf);
}

static esp_err_t api_printer_temp_get(httpd_req_t *req)
{
    char raw[64];
    int temp = -1;
    esp_err_t err = printer_drv_query_temperature(&temp, raw, sizeof(raw));
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }
    char buf[160];
    snprintf(buf, sizeof(buf), "{\"ok\":true,\"temperature_c\":%d,\"raw\":\"%s\"}", temp, raw);
    httpd_resp_set_type(req, "application/json");
    return httpd_resp_sendstr(req, buf);
}

static esp_err_t api_printer_baud_get(httpd_req_t *req)
{
    char raw[80];
    int printer_baud = -1;
    esp_err_t err = printer_drv_query_printer_baud(&printer_baud, raw, sizeof(raw));
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }
    char buf[192];
    snprintf(buf, sizeof(buf), "{\"ok\":true,\"esp32_baud\":%d,\"printer_baud\":%d,\"raw\":\"%s\"}",
             printer_drv_get_baud(), printer_baud, raw);
    httpd_resp_set_type(req, "application/json");
    return httpd_resp_sendstr(req, buf);
}

static esp_err_t api_printer_density_post(httpd_req_t *req)
{
    size_t body_len = 0; char *body = read_body(req, &body_len); (void)body_len;
    int density = -1;
    if (!body || !json_get_int(body, "density", &density) || density < 0 || density > 39) {
        free(body); send_json_error(req, "expected {\"density\":0..39}"); return ESP_FAIL;
    }
    esp_err_t err = printer_drv_set_density((uint8_t)density);
    free(body);
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }
    char buf[64]; snprintf(buf, sizeof(buf), "{\"ok\":true,\"density\":%d}", density);
    httpd_resp_set_type(req, "application/json"); return httpd_resp_sendstr(req, buf);
}

static bool web_speed_supported(int speed)
{
    static const int speeds[] = { 25, 30, 37, 50, 56, 62, 70, 80, 90, 100, 120, 150, 180, 200, 220 };
    for (size_t i = 0; i < sizeof(speeds) / sizeof(speeds[0]); i++) if (speeds[i] == speed) return true;
    return false;
}

static esp_err_t api_printer_speed_post(httpd_req_t *req)
{
    size_t body_len = 0; char *body = read_body(req, &body_len); (void)body_len;
    int speed = -1;
    if (!body || !json_get_int(body, "speed", &speed) || !web_speed_supported(speed)) {
        free(body); send_json_error(req, "expected valid {\"speed\":n}"); return ESP_FAIL;
    }
    esp_err_t err = printer_drv_set_speed((uint8_t)speed);
    free(body);
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }
    char buf[64]; snprintf(buf, sizeof(buf), "{\"ok\":true,\"speed\":%d}", speed);
    httpd_resp_set_type(req, "application/json"); return httpd_resp_sendstr(req, buf);
}

static esp_err_t api_printer_graphics_mode_post(httpd_req_t *req)
{
    size_t body_len = 0; char *body = read_body(req, &body_len); (void)body_len;
    int mode = -1;
    if (!body || !json_get_int(body, "mode", &mode) || (mode != 30 && mode != 31 && mode != 32)) {
        free(body); send_json_error(req, "expected {\"mode\":30|31|32}"); return ESP_FAIL;
    }
    esp_err_t err = printer_drv_set_graphics_mode((uint8_t)mode);
    free(body);
    if (err != ESP_OK) { send_json_error(req, esp_err_to_name(err)); return ESP_FAIL; }
    const char *desc = mode == 30 ? "BLE" : (mode == 31 ? "adaptive" : "constant");
    char buf[96]; snprintf(buf, sizeof(buf), "{\"ok\":true,\"mode\":%d,\"description\":\"%s\"}", mode, desc);
    httpd_resp_set_type(req, "application/json"); return httpd_resp_sendstr(req, buf);
}

/* ---- POST /api/print/bitmap ------------------------------------------ */

static esp_err_t api_print_bitmap_post(httpd_req_t *req)
{
    /* Expect headers: X-Width (pixels) and X-Height (pixels) */
    char hdr_val[16] = {0};
    uint16_t width = 0, height = 0;

    if (httpd_req_get_hdr_value_str(req, "X-Width", hdr_val, sizeof(hdr_val)) == ESP_OK) {
        width = (uint16_t)atoi(hdr_val);
    }
    if (httpd_req_get_hdr_value_str(req, "X-Height", hdr_val, sizeof(hdr_val)) == ESP_OK) {
        height = (uint16_t)atoi(hdr_val);
    }

    if (width == 0 || height == 0 || (width % 8) != 0) {
        send_json_error(req, "missing or invalid X-Width/X-Height headers");
        return ESP_FAIL;
    }

    size_t expected = (size_t)(width / 8) * height;
    if (req->content_len <= 0 || (size_t)req->content_len != expected) {
        char msg[128];
        snprintf(msg, sizeof(msg),
                 "content-length %ld != expected %zu",
                 (long)req->content_len, expected);
        send_json_error(req, msg);
        return ESP_FAIL;
    }

    /* Clear any stale printer state (ESC @ = initialize) so leftover
     * bytes from a previous partial command don't corrupt this print. */
    printer_drv_reset();

    /* Read the entire body into memory first.  Network receive gaps must not
     * occur inside a raster command's pixel payload.  The printer driver then
     * sends one complete GS v 0 raster command while UART CTS flow control lets
     * the printer pause TX when its input buffer or thermal engine is busy. */
    size_t body_len = 0;
    char *body = read_body(req, &body_len);
    if (!body || body_len != expected) {
        free(body);
        send_json_error(req, "failed to read full bitmap body");
        return ESP_FAIL;
    }

    esp_err_t err = printer_drv_print_bitmap(width, height,
                                             (const uint8_t *)body);
    free(body);

    if (err != ESP_OK) {
        send_json_error(req, "bitmap print failed");
        return ESP_FAIL;
    }

    /* Feed N lines after printing (default 3) so the page can be torn off cleanly */
    {
        char feed_hdr[8] = {0};
        uint8_t feed = 3;
        if (httpd_req_get_hdr_value_str(req, "X-Feed", feed_hdr, sizeof(feed_hdr)) == ESP_OK) {
            int f = atoi(feed_hdr);
            if (f >= 0 && f <= 20) feed = (uint8_t)f;
        }
        if (feed > 0) printer_drv_feed(feed);
    }

    send_json_ok(req);
    return ESP_OK;
}

/* ---- GET /almanach — serve Almanach Studio SPA ---------------------- */

static esp_err_t almanach_root_get(httpd_req_t *req)
{
    httpd_resp_set_type(req, "text/html; charset=utf-8");
    httpd_resp_set_hdr(req, "Cache-Control", "public, max-age=3600");
    size_t len = (size_t)(almanach_index_html_end - almanach_index_html_start);
    /* EMBED_TXTFILES appends a NUL — strip it */
    if (len > 0 && almanach_index_html_start[len - 1] == '\0') len--;
    return httpd_resp_send(req, (const char *)almanach_index_html_start, len);
}

static esp_err_t almanach_bundle_js_get(httpd_req_t *req)
{
    httpd_resp_set_type(req, "application/javascript; charset=utf-8");
    httpd_resp_set_hdr(req, "Cache-Control", "public, max-age=86400");
    size_t len = (size_t)(almanach_bundle_js_end - almanach_bundle_js_start);
    /* EMBED_TXTFILES appends a NUL — strip it */
    if (len > 0 && almanach_bundle_js_start[len - 1] == '\0') len--;
    return httpd_resp_send(req, (const char *)almanach_bundle_js_start, len);
}

/* ---- Server start/stop ----------------------------------------------- */

static const httpd_uri_t uri_root = {
    .uri = "/", .method = HTTP_GET, .handler = root_get,
};
static const httpd_uri_t uri_status = {
    .uri = "/api/status", .method = HTTP_GET, .handler = api_status_get,
};
static const httpd_uri_t uri_print_text = {
    .uri = "/api/print/text", .method = HTTP_POST, .handler = api_print_text_post,
};
static const httpd_uri_t uri_print_bitmap = {
    .uri = "/api/print/bitmap", .method = HTTP_POST, .handler = api_print_bitmap_post,
};
static const httpd_uri_t uri_printer_status = {
    .uri = "/api/printer/status", .method = HTTP_GET, .handler = api_printer_status_get,
};
static const httpd_uri_t uri_printer_temp = {
    .uri = "/api/printer/temp", .method = HTTP_GET, .handler = api_printer_temp_get,
};
static const httpd_uri_t uri_printer_baud = {
    .uri = "/api/printer/baud", .method = HTTP_GET, .handler = api_printer_baud_get,
};
static const httpd_uri_t uri_printer_density = {
    .uri = "/api/printer/density", .method = HTTP_POST, .handler = api_printer_density_post,
};
static const httpd_uri_t uri_printer_speed = {
    .uri = "/api/printer/speed", .method = HTTP_POST, .handler = api_printer_speed_post,
};
static const httpd_uri_t uri_printer_graphics_mode = {
    .uri = "/api/printer/graphics-mode", .method = HTTP_POST, .handler = api_printer_graphics_mode_post,
};

static const httpd_uri_t uri_almanach_root = {
    .uri = "/almanach", .method = HTTP_GET, .handler = almanach_root_get,
};
static const httpd_uri_t uri_almanach_bundle = {
    .uri = "/almanach/bundle.js", .method = HTTP_GET, .handler = almanach_bundle_js_get,
};

esp_err_t web_server_start(void)
{
    if (s_server) return ESP_OK;

    httpd_config_t config = HTTPD_DEFAULT_CONFIG();
    config.stack_size = 8192;
    config.max_uri_handlers = 20;

    esp_err_t err = httpd_start(&s_server, &config);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "httpd_start failed: %s", esp_err_to_name(err));
        return err;
    }

    httpd_register_uri_handler(s_server, &uri_root);
    httpd_register_uri_handler(s_server, &uri_status);
    httpd_register_uri_handler(s_server, &uri_print_text);
    httpd_register_uri_handler(s_server, &uri_print_bitmap);
    httpd_register_uri_handler(s_server, &uri_printer_status);
    httpd_register_uri_handler(s_server, &uri_printer_temp);
    httpd_register_uri_handler(s_server, &uri_printer_baud);
    httpd_register_uri_handler(s_server, &uri_printer_density);
    httpd_register_uri_handler(s_server, &uri_printer_speed);
    httpd_register_uri_handler(s_server, &uri_printer_graphics_mode);

    httpd_register_uri_handler(s_server, &uri_almanach_root);
    httpd_register_uri_handler(s_server, &uri_almanach_bundle);

    ESP_LOGI(TAG, "HTTP server started on port 80");
    return ESP_OK;
}

void web_server_stop(void)
{
    if (s_server) {
        httpd_stop(s_server);
        s_server = NULL;
        ESP_LOGI(TAG, "HTTP server stopped");
    }
}
