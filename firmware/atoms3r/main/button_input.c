/*
 * AtomS3R button handling for Almanach pairing mode.
 *
 * ISR only queues edge/timestamp information. The task performs debouncing,
 * long-hold detection, provisioning calls, NVS erasure, and reboot from normal
 * FreeRTOS task context.
 */

#include "button_input.h"

#include <stdbool.h>
#include <stdint.h>
#include <stdio.h>

#include "sdkconfig.h"

#include "freertos/FreeRTOS.h"
#include "freertos/queue.h"
#include "freertos/task.h"

#include "driver/gpio.h"
#include "esp_err.h"
#include "esp_log.h"
#include "esp_system.h"
#include "esp_timer.h"

#include "display_app.h"
#include "nvs_store.h"
#include "provisioning_mgr.h"
#include "wifi_mgr.h"

static const char *TAG = "button_input";

typedef struct {
    int64_t ts_us;
    int level;
} button_edge_t;

static QueueHandle_t s_button_q = NULL;
static bool s_started = false;

static bool button_level_is_pressed(int level)
{
#if CONFIG_ALMANACH_ATOMS3R_BUTTON_ACTIVE_LOW
    return level == 0;
#else
    return level != 0;
#endif
}

static int button_read_level(void)
{
    return gpio_get_level((gpio_num_t)CONFIG_ALMANACH_ATOMS3R_BUTTON_GPIO);
}

static void IRAM_ATTR button_isr(void *arg)
{
    (void)arg;
    if (!s_button_q) {
        return;
    }
    button_edge_t ev = {
        .ts_us = esp_timer_get_time(),
        .level = button_read_level(),
    };
    BaseType_t hp_task_woken = pdFALSE;
    xQueueSendFromISR(s_button_q, &ev, &hp_task_woken);
    if (hp_task_woken) {
        portYIELD_FROM_ISR();
    }
}

static void reset_provisioning_and_reboot(void)
{
    display_app_show_error("Reset WiFi", "Rebooting...");
    ESP_LOGW(TAG, "button reset hold reached; clearing WiFi/provisioning state");

    wifi_mgr_disconnect();

    esp_err_t err = nvs_store_erase_wifi();
    if (err != ESP_OK) {
        ESP_LOGW(TAG, "explicit WiFi credential erase failed: %s", esp_err_to_name(err));
    }

    err = provisioning_mgr_reset();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "provisioning reset failed: %s", esp_err_to_name(err));
        display_app_show_error("Prov reset", esp_err_to_name(err));
        return;
    }

    vTaskDelay(pdMS_TO_TICKS(250));
    esp_restart();
}

static void maybe_start_pairing(void)
{
    provisioning_status_t before = {0};
    (void)provisioning_mgr_get_status(&before);

    esp_err_t err = provisioning_mgr_start_force();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "BLE provisioning start from button failed: %s", esp_err_to_name(err));
        display_app_show_error("Pair failed", esp_err_to_name(err));
        return;
    }

    provisioning_status_t st = {0};
    provisioning_mgr_get_status(&st);
    display_status_t ds = {0};
    ds.provisioned = st.provisioned;
    ds.provisioning_running = st.running;
    ds.provisioning_client_connected = st.client_connected;
    ds.provisioning_security_ok = st.security_ok;
    snprintf(ds.service_name, sizeof(ds.service_name), "%s", st.service_name);
    snprintf(ds.pop, sizeof(ds.pop), "%s", st.pop);
    snprintf(ds.message, sizeof(ds.message), "%s", before.running ? "Pairing already on" : "Pairing on");
    display_app_show_status(&ds);
}

static void button_task(void *arg)
{
    (void)arg;

    bool was_pressed = button_level_is_pressed(button_read_level());
    int64_t press_start_us = was_pressed ? esp_timer_get_time() : 0;
    int64_t last_edge_us = 0;
    bool pairing_announced = false;
    bool reset_triggered = false;

    while (true) {
        button_edge_t ev = {0};
        (void)xQueueReceive(s_button_q, &ev, pdMS_TO_TICKS(100));

        const int64_t now_us = esp_timer_get_time();
        const bool pressed = button_level_is_pressed(button_read_level());

        if (ev.ts_us != 0) {
            const int64_t debounce_us = (int64_t)CONFIG_ALMANACH_ATOMS3R_BUTTON_DEBOUNCE_MS * 1000;
            if (last_edge_us != 0 && (ev.ts_us - last_edge_us) < debounce_us) {
                continue;
            }
            last_edge_us = ev.ts_us;
        }

        if (pressed && !was_pressed) {
            press_start_us = now_us;
            pairing_announced = false;
            reset_triggered = false;
            ESP_LOGI(TAG, "button press started");
        }

        if (pressed && press_start_us != 0) {
            const uint32_t held_ms = (uint32_t)((now_us - press_start_us) / 1000);

            if (held_ms >= (uint32_t)CONFIG_ALMANACH_ATOMS3R_PAIRING_HOLD_MS && !pairing_announced) {
                ESP_LOGI(TAG, "button pairing hold reached: %lu ms", (unsigned long)held_ms);
                display_app_show_pairing_hold(held_ms, CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS);
                maybe_start_pairing();
                pairing_announced = true;
            } else if (pairing_announced && !reset_triggered &&
                       held_ms < (uint32_t)CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS) {
                display_app_show_pairing_hold(held_ms, CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS);
            }

            if (held_ms >= (uint32_t)CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS && !reset_triggered) {
                reset_triggered = true;
                reset_provisioning_and_reboot();
            }
        }

        if (!pressed && was_pressed) {
            const uint32_t held_ms = press_start_us ? (uint32_t)((now_us - press_start_us) / 1000) : 0;
            ESP_LOGI(TAG, "button released after %lu ms", (unsigned long)held_ms);
            press_start_us = 0;
            pairing_announced = false;
            reset_triggered = false;
        }

        was_pressed = pressed;
    }
}

esp_err_t button_input_start(void)
{
    if (s_started) {
        return ESP_OK;
    }

    s_button_q = xQueueCreate(16, sizeof(button_edge_t));
    if (!s_button_q) {
        return ESP_ERR_NO_MEM;
    }

    const gpio_num_t pin = (gpio_num_t)CONFIG_ALMANACH_ATOMS3R_BUTTON_GPIO;
    gpio_config_t io = {
        .pin_bit_mask = 1ULL << (int)pin,
        .mode = GPIO_MODE_INPUT,
        .pull_up_en = GPIO_PULLUP_ENABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
        .intr_type = GPIO_INTR_ANYEDGE,
    };
    ESP_ERROR_CHECK(gpio_config(&io));

    esp_err_t err = gpio_install_isr_service(0);
    if (err != ESP_OK && err != ESP_ERR_INVALID_STATE) {
        return err;
    }
    ESP_ERROR_CHECK(gpio_isr_handler_add(pin, button_isr, NULL));

    ESP_LOGI(TAG, "button init: gpio=%d active_low=%d debounce_ms=%d pair_hold_ms=%d reset_hold_ms=%d",
             (int)pin,
#if CONFIG_ALMANACH_ATOMS3R_BUTTON_ACTIVE_LOW
             1,
#else
             0,
#endif
             CONFIG_ALMANACH_ATOMS3R_BUTTON_DEBOUNCE_MS,
             CONFIG_ALMANACH_ATOMS3R_PAIRING_HOLD_MS,
             CONFIG_ALMANACH_ATOMS3R_PAIRING_RESET_HOLD_MS);

    xTaskCreate(button_task, "button_pair", 4096, NULL, 8, NULL);
    s_started = true;
    return ESP_OK;
}
