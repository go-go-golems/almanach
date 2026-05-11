#!/usr/bin/env python3
"""Serve the standalone raster dither lab and proxy bitmap prints.

Why this exists:
- Opening raster-dither-lab.html from file:// and POSTing directly to
  http://192.168.1.242/api/print/bitmap hits browser CORS/private-network rules.
- Serving the page from localhost and POSTing to a same-origin localhost proxy
  avoids browser CORS, while this script forwards the raw bitmap request to the
  printer.

Usage:

    python scripts/01-serve-raster-dither-lab.py --port 18301 --printer http://192.168.1.242

Then open:

    http://localhost:18301/raster-dither-lab.html
"""

from __future__ import annotations

import argparse
import http.client
import mimetypes
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import urlparse

SCRIPT_DIR = Path(__file__).resolve().parent
TICKET_DIR = SCRIPT_DIR.parent
DEFAULT_ROOT = TICKET_DIR / "various"
DEFAULT_PRINTER = "http://192.168.1.242"
MAX_BITMAP_BODY_BYTES = 90 * 1024


class Handler(BaseHTTPRequestHandler):
    root: Path
    printer_base: str

    def log_message(self, fmt: str, *args) -> None:  # noqa: A003
        print(f"[{self.log_date_time_string()}] {self.address_string()} {fmt % args}")

    def _send_cors(self) -> None:
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
        self.send_header("Access-Control-Allow-Headers", "Content-Type, X-Width, X-Height, X-Feed")

    def do_OPTIONS(self) -> None:  # noqa: N802
        self.send_response(204)
        self._send_cors()
        self.end_headers()

    def do_GET(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        if path in ("/", ""):
            path = "/raster-dither-lab.html"
        rel = path.lstrip("/")
        file_path = (self.root / rel).resolve()
        try:
            file_path.relative_to(self.root.resolve())
        except ValueError:
            self.send_error(403)
            return
        if not file_path.is_file():
            self.send_error(404)
            return
        data = file_path.read_bytes()
        content_type = mimetypes.guess_type(str(file_path))[0] or "application/octet-stream"
        self.send_response(200)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self._send_cors()
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self) -> None:  # noqa: N802
        path = urlparse(self.path).path
        if path != "/api/print/bitmap":
            self.send_error(404)
            return
        try:
            length = int(self.headers.get("Content-Length", "0"))
        except ValueError:
            length = 0
        if length <= 0 or length > MAX_BITMAP_BODY_BYTES:
            self.send_response(413)
            self.send_header("Content-Type", "application/json")
            self._send_cors()
            self.end_headers()
            self.wfile.write(f'{{"ok":false,"error":"bitmap too large or empty: {length}"}}'.encode())
            return
        body = self.rfile.read(length)
        printer = urlparse(self.printer_base)
        conn_cls = http.client.HTTPSConnection if printer.scheme == "https" else http.client.HTTPConnection
        host = printer.hostname or "192.168.1.242"
        port = printer.port or (443 if printer.scheme == "https" else 80)
        conn = conn_cls(host, port, timeout=45)
        headers = {
            "Content-Type": "application/octet-stream",
            "Content-Length": str(len(body)),
            "X-Width": self.headers.get("X-Width", ""),
            "X-Height": self.headers.get("X-Height", ""),
            "X-Feed": self.headers.get("X-Feed", "0"),
        }
        try:
            conn.request("POST", "/api/print/bitmap", body=body, headers=headers)
            resp = conn.getresponse()
            resp_body = resp.read()
        except Exception as e:  # pragma: no cover - operator tool
            self.send_response(502)
            self.send_header("Content-Type", "application/json")
            self._send_cors()
            self.end_headers()
            self.wfile.write((f'{{"ok":false,"error":"proxy failed: {e}"}}').encode())
            return
        finally:
            conn.close()
        self.send_response(resp.status)
        self.send_header("Content-Type", resp.getheader("Content-Type") or "application/json")
        self.send_header("Content-Length", str(len(resp_body)))
        self._send_cors()
        self.end_headers()
        self.wfile.write(resp_body)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="127.0.0.1")
    parser.add_argument("--port", type=int, default=18301)
    parser.add_argument("--root", type=Path, default=DEFAULT_ROOT)
    parser.add_argument("--printer", default=os.environ.get("ALMANACH_PRINTER_URL", DEFAULT_PRINTER))
    args = parser.parse_args()

    Handler.root = args.root
    Handler.printer_base = args.printer.rstrip("/")
    server = ThreadingHTTPServer((args.host, args.port), Handler)
    print(f"Serving raster lab: http://localhost:{args.port}/raster-dither-lab.html")
    print(f"Proxying /api/print/bitmap -> {Handler.printer_base}/api/print/bitmap")
    server.serve_forever()


if __name__ == "__main__":
    main()
