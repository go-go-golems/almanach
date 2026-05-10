# Bitmap Stripes, Pauses, CTS, and Software Flow Control

This document records the bitmap stripe investigation and the transport/thermal/power hypotheses that emerged while building SToMS3R.

## The observed failure modes

We saw several distinct behaviors that should not be conflated:

1. **No print at all** — initially caused by wrong GPIO mapping and straight-through cable behavior.
2. **Text printing `:` instead of the requested string** — caused by a JSON parser bug in `/api/print/text`.
3. **Bitmap stripes from chunked HTTP-to-UART streaming** — caused by network receive gaps inside the raster payload.
4. **Regular, more frequent stripes after 5-row banding** — caused by visible seams at artificial raster-band boundaries.
5. **Pause-synchronized stripes** — likely related to printer input-buffer, CTS/flow-control, thermal throttling, or power supply sag.

Each of these needed a different fix.

## Correct GPIO and cable behavior

The K118 kit was designed for the ATOM Lite bottom header:

```text
ATOM Lite: TX=GPIO23, RX=GPIO33, CTS=GPIO19
```

On the AtomS3R Lite, the same physical header positions map to:

```text
TX header position  -> GPIO8
RX header position  -> GPIO7
CTS header position -> GPIO6
```

The K118 cable appears to be straight-through. Therefore the firmware defaults to software TX/RX swapping:

```text
UART TX uses GPIO7
UART RX uses GPIO8
CTS uses GPIO6
```

This is why `printer_swap` exists and why swap defaults on.

## Why the original Arduino bitmap path mattered

The original `ATOM_PRINTER::printBMP()` does this:

```cpp
_serial->write(buffer, sizeof(PRINTER_BMP_CMD));
len = xdot * ydot;
while (len) {
    _serial->write(*buffer++);
    len--;
}
```

The important part is not that it writes one byte at a time as a requirement. The important part is that Arduino `HardwareSerial::write()` queues bytes into a UART ring buffer and the UART ISR drains them continuously. The original firmware does not interleave TCP reads with printer writes.

The original web firmware receives the uploaded bitmap into a buffer first and only then calls `printer.printBMP(...)`.

## Why direct HTTP chunk streaming caused stripes

The first ESP-IDF web bitmap path did this:

```text
read 128/512 bytes from HTTP
write chunk to UART
read next HTTP chunk
write chunk to UART
...
```

At 9600 baud, printer data timing is slow enough that gaps matter. A TCP receive call between UART writes can create a visible pause inside one `GS v 0` raster command's pixel payload. The printer can interpret or experience that as a boundary, overrun, or lost-data condition, producing horizontal stripes.

The first fix was correct:

```text
read entire HTTP body into heap memory
then print from local memory
```

This invariant remains important:

> Never allow arbitrary network receive gaps inside a raster command's pixel payload.

## Why 5-row raster bands made stripes regular

A later experiment printed a large image as many small complete `GS v 0` raster commands:

```text
5 rows per band
48 bytes per row
240 bytes per band
50ms delay between bands
```

This was based on the common 256-byte-buffer heuristic used by small serial thermal printer libraries when no flow-control signal is trusted.

But testing showed the stripes became twice as frequent and regular. That strongly suggests the band boundaries themselves were visible. Even if pauses between complete raster commands are legal, this printer/paper/mechanism can show seams at those boundaries.

Conclusion:

> Banded raster printing is a diagnostic fallback, not the default print path.

## Current preferred bitmap strategy

The current preferred strategy is:

```text
1. Browser does all image processing.
2. ESP32 reads full HTTP body into heap memory.
3. ESP32 sends one complete GS v 0 raster command.
4. ESP32 UART uses CTS hardware flow control so the printer can pause TX.
```

The relevant UART config is:

```c
.flow_ctrl = UART_HW_FLOWCTRL_CTS
```

and CTS is mapped to GPIO6:

```c
uart_set_pin(UART_NUM_1, tx, rx, UART_PIN_NO_CHANGE, GPIO6);
```

## CTS flow control

Hardware CTS flow control uses a dedicated signal line. The printer indicates whether it is ready to receive data. The ESP32 UART peripheral pauses TX automatically when CTS says the receiver is not ready.

Conceptually:

```text
if CTS says ready:
    UART sends bytes
else:
    UART pauses TX in hardware
```

This is better than application-level delay because it is synchronized with the receiver's actual busy/ready signal, assuming the K118 CTS line is correctly wired and uses the polarity expected by the ESP32 UART hardware.

If printing hangs immediately after enabling CTS, likely causes are:

- CTS polarity mismatch
- CTS not connected the way we think
- printer does not drive CTS in TTL UART mode
- straight-through/crossover confusion on the CTS line is irrelevant but TX/RX may still be wrong

In that case, add a runtime flow-mode command to toggle CTS without reflashing.

## Software flow control

The spec includes:

```text
ESC ## SFFC n
Hex: 1B 23 23 53 46 46 43 n

n = 0: disable software flow control
n = 1: enable software flow control
```

In serial-printer terminology, software flow control usually means XON/XOFF:

```text
XOFF = 0x13  // receiver says: stop sending
XON  = 0x11  // receiver says: resume sending
```

Unlike CTS, this uses the same serial data lines:

```text
ESP32 TX  -> printer RX: bitmap bytes
ESP32 RX  <- printer TX: XOFF / XON bytes
```

### Important: enabling it on the printer is not enough

The host must actively honor the control bytes. A huge call like:

```c
uart_write_bytes(UART_NUM_1, pixels, len);
```

cannot respond quickly to XOFF because the data may already be queued into the UART driver. A software-flow-aware sender must write in small chunks and poll RX between chunks.

Pseudo-code:

```c
static bool paused = false;

static void poll_flow_control(void) {
    uint8_t b;
    while (uart_read_bytes(UART_NUM_1, &b, 1, 0) == 1) {
        if (b == 0x13) paused = true;   // XOFF
        if (b == 0x11) paused = false;  // XON
    }
}

while (off < len) {
    poll_flow_control();
    while (paused) {
        vTaskDelay(pdMS_TO_TICKS(5));
        poll_flow_control();
    }

    size_t chunk = MIN(32, len - off);
    uart_write_bytes(UART_NUM_1, data + off, chunk);
    off += chunk;
}
```

### Recommended software-flow implementation path

1. Add `printer_softflow on|off` that only sends `ESC ## SFFC n`.
2. Add RX logging during bitmap prints for `0x11` and `0x13`.
3. Print dense bitmap patterns and see whether the printer emits XON/XOFF.
4. Only if XON/XOFF is observed, add a chunked software-flow-aware bitmap sender.
5. Compare flow modes:

```text
none
cts
xonxoff
both
```

## Thermal and power hypotheses

The K118 product docs state:

```text
DC 12V, 2.5A recommended
power supply capability directly affects print display quality
```

Dense bitmaps, especially full black, are not normal workloads. They stress:

- print-head current
- supply voltage
- thermal management
- internal input-buffer timing
- paper movement consistency

Use `fullblack` only as a stress test. Prefer `graylevels`, `diagonal`, and real dithered images for useful print-quality evaluation.

## Diagnostics now available

Current firmware exposes:

```text
printer_status          // GS a n: buffer_full, overheat, paper, cover
printer_temp            // GS g 6
printer_get_baud        // GS g 7
printer_density <0-39>  // ESC ## STDP
printer_speed <n>       // ESC ## STSP
printer_graphics_mode   // ESC ## SPSM
```

Recommended stripe test procedure:

```text
printer_status
printer_temp
printer_get_baud
printer_density 20
printer_speed 80
printer_graphics_mode 31
# print graylevels
printer_status
printer_temp
```

Then compare:

```text
printer_graphics_mode 32
# print same graylevels
printer_status
printer_temp
```

Watch especially:

- `buffer_full`
- `overheated`
- temperature before/after
- whether adaptive graphics mode changes stripe spacing
- whether lower density reduces pauses

## Practical conclusions

- Full-body buffering before UART remains correct.
- Artificial raster banding created visible seams on this printer and should not be default.
- CTS is the right first flow-control mechanism because it is hardware-level and does not require chunked application writes.
- Software flow control may help, but only if the printer actually emits XON/XOFF and the ESP32 sender is changed to honor it.
- Density, speed, and graphics mode are first-class debugging knobs, not optional polish.
