Import("env")

import os
from pathlib import Path


def _strip_wrapping_quotes(value: str) -> str:
    if len(value) >= 2 and (
        (value[0] == '"' and value[-1] == '"')
        or (value[0] == "'" and value[-1] == "'")
    ):
        return value[1:-1]
    return value


def _parse_env_file(path: Path) -> dict:
    parsed = {}
    for raw_line in path.read_text(encoding="utf-8").splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#"):
            continue

        if line.startswith("export "):
            line = line[len("export ") :].strip()

        if "=" not in line:
            continue

        key, value = line.split("=", 1)
        key = key.strip()
        value = value.strip()

        # Keep inline comments only when value is quoted.
        if value and value[0] not in ('"', "'"):
            comment_marker = value.find(" #")
            if comment_marker >= 0:
                value = value[:comment_marker].strip()

        parsed[key] = _strip_wrapping_quotes(value)

    return parsed


def _read_env_values(project_dir: Path) -> dict:
    combined = {}
    candidates = [
        project_dir.parent / ".env",
        project_dir.parent / ".env.local",
        project_dir / ".env",
        project_dir / ".env.local",
    ]

    for candidate in candidates:
        if candidate.is_file():
            combined.update(_parse_env_file(candidate))

    return combined


def _resolve_value(key: str, env_values: dict) -> str:
    process_value = os.getenv(key)
    if process_value is not None and process_value != "":
        return process_value
    return env_values.get(key, "")


def _parse_int_like(value: str):
    lowered = value.strip().lower()
    if lowered in ("1", "true", "yes", "on"):
        return 1
    if lowered in ("0", "false", "no", "off"):
        return 0

    try:
        return int(value)
    except ValueError:
        return None


def _append_cpp_string_define(name: str, value: str):
    if value == "":
        return False

    escaped = value.replace("\\", "\\\\").replace('"', '\\"')
    env.Append(CPPDEFINES=[(name, f'\\"{escaped}\\"')])
    return True


pio_env = env.subst("$PIOENV")
if pio_env != "t-a7670-wokwi":
    print(f"[wokwi-env] Skipping .env load for environment: {pio_env}")
else:
    project_dir = Path(env["PROJECT_DIR"])
    env_values = _read_env_values(project_dir)

    string_keys = [
        "WOKWI_DEFAULT_WIFI_SSID",
        "WOKWI_DEFAULT_WIFI_PASS",
        "WOKWI_DEFAULT_MQTT_HOST",
        "WOKWI_DEFAULT_MQTT_USER",
        "WOKWI_DEFAULT_MQTT_PASS",
    ]

    int_keys = [
        "MQTT_USE_TLS",
        "MQTT_SKIP_CERT_VERIFY",
        "MQTT_PORT",
        "MQTT_PORT_TLS",
    ]

    applied = []

    for key in string_keys:
        raw_value = _resolve_value(key, env_values)
        if _append_cpp_string_define(key, raw_value):
            applied.append(key)

    for key in int_keys:
        raw_value = _resolve_value(key, env_values)
        if raw_value == "":
            continue

        parsed = _parse_int_like(raw_value)
        if parsed is None:
            print(f"[wokwi-env] Ignoring invalid numeric value for {key}")
            continue

        env.Append(CPPDEFINES=[(key, parsed)])
        applied.append(key)

    if applied:
        print("[wokwi-env] Applied .env overrides: " + ", ".join(applied))
    else:
        print("[wokwi-env] No .env overrides found, using firmware defaults")
