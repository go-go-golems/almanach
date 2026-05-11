#!/usr/bin/env python3
"""devctl plugin for the standalone Almanach repository.

The plugin keeps devctl responsible for orchestration/supervision and keeps this
file focused on repo-specific facts: how to build the web bundle, where the Go
binary lives, which port to use, and which helper commands are useful during
local development.
"""

import json
import os
import shutil
import subprocess
import sys
import time
from pathlib import Path


def emit(obj):
    sys.stdout.write(json.dumps(obj) + "\n")
    sys.stdout.flush()


def log(msg):
    sys.stderr.write("[almanach] " + msg + "\n")
    sys.stderr.flush()


def run(cmd, *, cwd, env=None, timeout=None):
    merged_env = os.environ.copy()
    if env:
        merged_env.update(env)
    log(f"run: {' '.join(cmd)} (cwd={cwd})")
    return subprocess.run(
        cmd,
        cwd=cwd,
        env=merged_env,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        timeout=timeout,
    )


def deadline_timeout(ctx, fallback_seconds=300):
    deadline_ms = int(ctx.get("deadline_ms") or 0)
    if deadline_ms <= 0:
        return fallback_seconds
    # devctl versions may pass either an absolute epoch-ms deadline or a
    # duration in milliseconds. Treat large values as epoch-ms, smaller values
    # as duration-ms so dynamic commands don't accidentally get a 1s timeout.
    if deadline_ms > 1_000_000_000_000:
        now_ms = int(time.time() * 1000)
        remaining = max(1, int((deadline_ms - now_ms) / 1000))
    else:
        remaining = max(1, int(deadline_ms / 1000))
    return min(fallback_seconds, remaining)


def step_result(name, started, proc=None, skipped=False, command=None):
    result = {
        "name": name,
        "ok": True if skipped else (proc is not None and proc.returncode == 0),
        "duration_ms": int((time.time() - started) * 1000),
    }
    if command:
        result["command"] = " ".join(command)
    if skipped:
        result["skipped"] = True
    if proc is not None and proc.returncode != 0:
        result["exit_code"] = proc.returncode
        result["stderr_tail"] = proc.stderr[-2000:]
    return result


PLUGIN_DIR = Path(__file__).resolve().parent
DEFAULT_REPO_ROOT = PLUGIN_DIR.parent
DEFAULT_PORT = 8199
DEFAULT_PRINTER_IP = ""


emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "almanach",
    "capabilities": {
        "ops": ["config.mutate", "validate.run", "build.run", "launch.plan", "command.run"],
        "commands": [
            {"name": "health", "help": "Check the Almanach render service /health endpoint"},
            {"name": "render", "help": "Render a layout to a PNG; args: [layout] [out]"},
            {"name": "inspect", "help": "Inspect render DOM metrics; args: [layout]"},
            {"name": "print", "help": "Render and print a layout; args: [layout]"},
            {"name": "build", "help": "Build web assets and the embedded Almanach render-service binary"},
            {"name": "build-web", "help": "Build Almanach Studio and refresh embedded Go assets"},
            {"name": "sync-firmware-web", "help": "Copy built web assets into firmware/atoms3r embedded assets"},
            {"name": "firmware-build", "help": "Build the AtomS3R ESP-IDF firmware"},
        ],
    },
})


for raw_line in sys.stdin:
    line = raw_line.strip()
    if not line:
        continue

    req = json.loads(line)
    rid = req.get("request_id", "")
    op = req.get("op", "")
    ctx = req.get("ctx", {}) or {}
    inp = req.get("input", {}) or {}

    dry_run = bool(ctx.get("dry_run", False))
    repo_root = Path(ctx.get("repo_root") or DEFAULT_REPO_ROOT).resolve()
    var_dir = repo_root / "var" / "devctl"
    binary = var_dir / "almanach-render-service"
    web_dist = repo_root / "web" / "dist"
    default_layout = repo_root / "examples" / "layouts" / "01-minimal.yaml"
    port = os.environ.get("ALMANACH_PORT", str(DEFAULT_PORT))
    printer_ip = os.environ.get("ALMANACH_PRINTER_IP", DEFAULT_PRINTER_IP)

    try:
        if op == "config.mutate":
            emit({
                "type": "response",
                "request_id": rid,
                "ok": True,
                "output": {
                    "config_patch": {
                        "set": {
                            "services.render.port": int(port),
                            "services.render.url": f"http://127.0.0.1:{port}",
                            "artifacts.render.binary": str(binary),
                            "artifacts.web.dist": str(web_dist),
                            "env.ALMANACH_PORT": port,
                            "env.ALMANACH_PRINTER_IP": printer_ip,
                        },
                        "unset": [],
                    }
                },
            })

        elif op == "validate.run":
            errors = []
            warnings = []
            for tool in ["go", "python3"]:
                if not shutil.which(tool):
                    errors.append({"code": "E_MISSING_TOOL", "message": f"{tool} not found on PATH"})
            if not (repo_root / "cmd" / "almanach-render-service" / "main.go").exists():
                errors.append({"code": "E_MISSING_ENTRYPOINT", "message": "cmd/almanach-render-service/main.go is missing"})
            if not (repo_root / "cmd" / "build-web" / "main.go").exists():
                errors.append({"code": "E_MISSING_BUILD_WEB", "message": "cmd/build-web/main.go is missing"})
            if not (repo_root / "web" / "package.json").exists():
                errors.append({"code": "E_MISSING_WEB", "message": "web/package.json is missing"})
            if not shutil.which("pnpm") and os.environ.get("ALMANACH_BUILD_WEB_LOCAL", "1") != "0":
                warnings.append({"code": "W_NO_PNPM", "message": "pnpm not found; build-web local fallback will fail unless Dagger/Docker is used"})
            if not any(shutil.which(c) for c in ["chromium-browser", "chromium", "google-chrome", "chrome"]) and not os.environ.get("ALMANACH_CHROME_PATH"):
                warnings.append({"code": "W_NO_CHROME", "message": "Chrome/Chromium not found. Set ALMANACH_CHROME_PATH for rendering."})
            if not printer_ip:
                warnings.append({"code": "W_NO_PRINTER_IP", "message": "ALMANACH_PRINTER_IP not set; print helpers need --printer-ip/env later."})

            emit({
                "type": "response",
                "request_id": rid,
                "ok": True,
                "output": {"valid": len(errors) == 0, "errors": errors, "warnings": warnings},
            })

        elif op == "build.run":
            timeout = deadline_timeout(ctx)
            steps = []
            artifacts = {"render-binary": str(binary), "web-dist": str(web_dist)}

            if dry_run:
                for name, command in [
                    ("build-web", ["go", "run", "./cmd/build-web"]),
                    ("build-render-service", ["go", "build", "-tags", "embed", "-o", str(binary), "./cmd/almanach-render-service"]),
                ]:
                    started = time.time()
                    steps.append(step_result(name, started, skipped=True, command=command))
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"steps": steps, "artifacts": artifacts}})
                continue

            var_dir.mkdir(parents=True, exist_ok=True)

            build_web_env = {}
            if os.environ.get("ALMANACH_BUILD_WEB_LOCAL", "1") != "0":
                build_web_env["BUILD_WEB_LOCAL"] = "1"
            command = ["go", "run", "./cmd/build-web"]
            started = time.time()
            proc = run(command, cwd=repo_root, env=build_web_env, timeout=timeout)
            if proc.stdout:
                log(proc.stdout.rstrip())
            if proc.stderr:
                log(proc.stderr.rstrip())
            steps.append(step_result("build-web", started, proc=proc, command=command))
            if proc.returncode != 0:
                emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_BUILD_WEB", "message": proc.stderr[-2000:]}, "output": {"steps": steps}})
                continue

            command = ["go", "build", "-tags", "embed", "-o", str(binary), "./cmd/almanach-render-service"]
            started = time.time()
            proc = run(command, cwd=repo_root, timeout=timeout)
            if proc.stdout:
                log(proc.stdout.rstrip())
            if proc.stderr:
                log(proc.stderr.rstrip())
            steps.append(step_result("build-render-service", started, proc=proc, command=command))
            if proc.returncode != 0:
                emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_BUILD_GO", "message": proc.stderr[-2000:]}, "output": {"steps": steps}})
                continue

            emit({"type": "response", "request_id": rid, "ok": True, "output": {"steps": steps, "artifacts": artifacts}})

        elif op == "launch.plan":
            chrome_path = os.environ.get("ALMANACH_CHROME_PATH", "")
            if not chrome_path:
                for candidate in ["chromium-browser", "chromium", "google-chrome", "chrome"]:
                    found = shutil.which(candidate)
                    if found:
                        chrome_path = found
                        break

            env = {
                "ALMANACH_PORT": port,
                "ALMANACH_WEB_DIR": str(web_dist),
                "ALMANACH_PRINTER_IP": printer_ip,
            }
            if chrome_path:
                env["ALMANACH_CHROME_PATH"] = chrome_path

            command = [str(binary), "serve"]
            if not binary.exists():
                # `devctl plan` can run before `devctl up` has built artifacts.
                # Keep the plan readable and still executable for ad-hoc starts.
                command = ["go", "run", "./cmd/almanach-render-service", "serve"]

            emit({
                "type": "response",
                "request_id": rid,
                "ok": True,
                "output": {
                    "services": [
                        {
                            "name": "render",
                            "cwd": str(repo_root),
                            "command": command,
                            "env": env,
                            "health": {"type": "http", "url": f"http://127.0.0.1:{port}/health", "timeout_ms": 30000},
                        }
                    ]
                },
            })

        elif op == "command.run":
            name = inp.get("name", "")
            argv = inp.get("argv", []) or []
            timeout = deadline_timeout(ctx, fallback_seconds=180)
            base_url = f"http://127.0.0.1:{port}"
            cli = [str(binary)] if binary.exists() else ["go", "run", "./cmd/almanach-render-service"]

            if name == "health":
                proc = run(["curl", "-fsS", f"{base_url}/health"], cwd=repo_root, timeout=30)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode}})

            elif name == "render":
                layout = argv[0] if len(argv) >= 1 else str(default_layout)
                out = argv[1] if len(argv) >= 2 else "/tmp/almanach-render.png"
                command = cli + ["render", "--layout", layout, "--out", out, "--web-dir", str(web_dist)]
                proc = run(command, cwd=repo_root, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode, "artifacts": {"render-png": out}}})

            elif name == "inspect":
                layout = argv[0] if argv else str(default_layout)
                command = cli + ["inspect", "--layout", layout, "--web-dir", str(web_dist), "--output", "yaml"]
                proc = run(command, cwd=repo_root, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode}})

            elif name == "print":
                layout = argv[0] if argv else str(default_layout)
                command = cli + ["print", "--layout", layout, "--web-dir", str(web_dist)]
                if printer_ip:
                    command.extend(["--printer-ip", printer_ip])
                proc = run(command, cwd=repo_root, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode}})

            elif name == "build":
                var_dir.mkdir(parents=True, exist_ok=True)
                env = {"BUILD_WEB_LOCAL": "1"} if os.environ.get("ALMANACH_BUILD_WEB_LOCAL", "1") != "0" else {}
                build_web = ["go", "run", "./cmd/build-web"]
                proc = run(build_web, cwd=repo_root, env=env, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                if proc.returncode != 0:
                    emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode}})
                    continue
                build_go = ["go", "build", "-tags", "embed", "-o", str(binary), "./cmd/almanach-render-service"]
                proc = run(build_go, cwd=repo_root, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode, "artifacts": {"render-binary": str(binary), "web-dist": str(web_dist)}}})

            elif name == "build-web":
                command = ["go", "run", "./cmd/build-web"]
                env = {"BUILD_WEB_LOCAL": "1"} if os.environ.get("ALMANACH_BUILD_WEB_LOCAL", "1") != "0" else {}
                proc = run(command, cwd=repo_root, env=env, timeout=timeout)
                if proc.stdout:
                    log(proc.stdout.rstrip())
                if proc.stderr:
                    log(proc.stderr.rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode, "artifacts": {"web-dist": str(web_dist)}}})

            elif name == "sync-firmware-web":
                src_html = repo_root / "web" / "dist" / "index.html"
                src_js = repo_root / "web" / "dist" / "almanach-bundle.js"
                dst_dir = repo_root / "firmware" / "atoms3r" / "main" / "assets" / "almanach"
                if not src_html.exists() or not src_js.exists():
                    emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": 1, "message": "web/dist assets missing; run devctl build-web first"}})
                    continue
                dst_dir.mkdir(parents=True, exist_ok=True)
                shutil.copy2(src_html, dst_dir / "almanach.html")
                shutil.copy2(src_js, dst_dir / "almanach-bundle.js")
                log(f"synced {src_html} and {src_js} to {dst_dir}")
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": 0, "artifacts": {"firmware-assets": str(dst_dir)}}})

            elif name == "firmware-build":
                command = ["./build.sh", "/dev/ttyACM0", "build"]
                proc = run(command, cwd=repo_root / "firmware" / "atoms3r", timeout=timeout)
                if proc.stdout:
                    log(proc.stdout[-4000:].rstrip())
                if proc.stderr:
                    log(proc.stderr[-4000:].rstrip())
                emit({"type": "response", "request_id": rid, "ok": True, "output": {"exit_code": proc.returncode}})

            else:
                emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_UNKNOWN_COMMAND", "message": f"unknown command: {name}"}})

        else:
            emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_UNSUPPORTED", "message": f"unsupported op: {op}"}})

    except Exception as exc:
        log(f"error: {exc}")
        emit({"type": "response", "request_id": rid, "ok": False, "error": {"code": "E_INTERNAL", "message": str(exc)}})
