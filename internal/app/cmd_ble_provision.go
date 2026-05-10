package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
)

type BLEProvisionCommand struct {
	*cmds.CommandDescription
}

type BLEProvisionSettings struct {
	Action       string `glazed:"action"`
	ServiceName  string `glazed:"service-name"`
	SSID         string `glazed:"ssid"`
	Passphrase   string `glazed:"passphrase"`
	Pop          string `glazed:"pop"`
	SecVer       int    `glazed:"sec-ver"`
	ProtoVer     string `glazed:"proto-ver"`
	IDFPath      string `glazed:"idf-path"`
	Python       string `glazed:"python"`
	EspProv      string `glazed:"esp-prov"`
	Verbose      bool   `glazed:"verbose"`
	DryRun       bool   `glazed:"dry-run"`
	Timeout      int    `glazed:"timeout"`
	InstallHints bool   `glazed:"install-hints"`
}

func newBLEProvisionCommand() (*BLEProvisionCommand, error) {
	glazedSection, err := settings.NewGlazedSchema()
	if err != nil {
		return nil, err
	}
	commandSettingsSection, err := cli.NewCommandSettingsSection()
	if err != nil {
		return nil, err
	}

	desc := cmds.NewCommandDescription(
		"ble-provision",
		cmds.WithShort("Provision an Almanach AtomS3R over ESP-IDF BLE provisioning"),
		cmds.WithLong(`Provision or reset an Almanach AtomS3R using ESP-IDF's BLE protocomm provisioning client.

This command is a Linux-friendly Go/Glazed wrapper around ESP-IDF's esp_prov.py. It keeps the
workflow inside the Almanach binary while reusing Espressif's maintained BLE, protobuf, and
Security 1 implementation.

Examples:
  almanach-render-service ble-provision --action provision --service-name ALM_0F2320 --pop alm-0f2320 --ssid MyWifi --passphrase secret
  almanach-render-service ble-provision --action reset --service-name ALM_0F2320 --pop alm-0f2320
  almanach-render-service ble-provision --action version --service-name ALM_0F2320 --pop alm-0f2320 --dry-run

Notes:
  - Linux BLE access normally requires BlueZ and permissions for the active user.
  - The ESP-IDF Python environment must include bleak, protobuf, and cryptography.
  - Use --install-hints if dependency import errors appear.
`),
		cmds.WithFlags(bleProvisionFields()...),
		cmds.WithSections(glazedSection, commandSettingsSection),
	)
	return &BLEProvisionCommand{CommandDescription: desc}, nil
}

func bleProvisionFields() []*fields.Definition {
	return []*fields.Definition{
		fields.New("action", fields.TypeChoice, fields.WithDefault("provision"), fields.WithChoices("provision", "reset", "reprov", "version"), fields.WithHelp("Provisioning operation to run")),
		fields.New("service-name", fields.TypeString, fields.WithDefault(""), fields.WithHelp("BLE device/provisioning service name, e.g. ALM_0F2320")),
		fields.New("ssid", fields.TypeString, fields.WithDefault(""), fields.WithHelp("WiFi SSID to configure; required for action=provision")),
		fields.New("passphrase", fields.TypeString, fields.WithDefault(""), fields.WithHelp("WiFi passphrase; if omitted for action=provision, read one line from stdin")),
		fields.New("pop", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Security 1 proof-of-possession, e.g. alm-0f2320")),
		fields.New("sec-ver", fields.TypeInteger, fields.WithDefault(1), fields.WithHelp("ESP protocomm security version; Almanach firmware uses Security 1")),
		fields.New("proto-ver", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Optional protocol version string to verify via proto-ver before provisioning")),
		fields.New("idf-path", fields.TypeString, fields.WithDefault(defaultIDFPath()), fields.WithHelp("ESP-IDF root; used to locate tools/esp_prov/esp_prov.py and set IDF_PATH")),
		fields.New("python", fields.TypeString, fields.WithDefault(defaultIDFPython()), fields.WithHelp("Python interpreter for esp_prov.py; defaults to the ESP-IDF 5.4 Python env when present")),
		fields.New("esp-prov", fields.TypeString, fields.WithDefault(""), fields.WithHelp("Explicit esp_prov.py path; defaults to $IDF_PATH/tools/esp_prov/esp_prov.py")),
		fields.New("verbose", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Pass --verbose to esp_prov.py")),
		fields.New("dry-run", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Print the resolved command without executing it")),
		fields.New("timeout", fields.TypeInteger, fields.WithDefault(120), fields.WithHelp("Timeout in seconds for the esp_prov.py subprocess")),
		fields.New("install-hints", fields.TypeBool, fields.WithDefault(false), fields.WithHelp("Print Linux/ESP-IDF dependency installation hints before running")),
	}
}

func (c *BLEProvisionCommand) RunIntoGlazeProcessor(ctx context.Context, vals *values.Values, gp middlewares.Processor) error {
	s := &BLEProvisionSettings{}
	if err := vals.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	if s.Action == "provision" && s.SSID == "" {
		return errors.New("--ssid is required for action=provision")
	}
	if s.ServiceName == "" {
		return errors.New("--service-name is required; use the value printed by firmware prov_status, e.g. ALM_0F2320")
	}
	if s.Timeout <= 0 {
		return errors.New("--timeout must be greater than zero")
	}

	passphrase := s.Passphrase
	readPassphrase := false
	if s.Action == "provision" && passphrase == "" {
		var err error
		passphrase, err = readLineFromStdin("WiFi passphrase: ")
		if err != nil {
			return err
		}
		readPassphrase = true
	}

	pythonPath := s.Python
	if pythonPath == "" {
		pythonPath = "python3"
	}
	espProvPath := s.EspProv
	if espProvPath == "" {
		espProvPath = filepath.Join(s.IDFPath, "tools", "esp_prov", "esp_prov.py")
	}

	args, displayArgs, err := buildBLEProvisionPythonArgs(s, espProvPath, passphrase)
	if err != nil {
		return err
	}

	if s.InstallHints {
		fmt.Fprintln(os.Stderr, bleProvisionInstallHints(s.IDFPath))
	}

	if s.DryRun {
		fmt.Fprintf(os.Stderr, "IDF_PATH=%s %s %s\n", s.IDFPath, pythonPath, strings.Join(displayArgs, " "))
		return gp.AddRow(ctx, types.NewRow(
			types.MRP("action", s.Action),
			types.MRP("service_name", s.ServiceName),
			types.MRP("ssid", s.SSID),
			types.MRP("pop", s.Pop),
			types.MRP("idf_path", s.IDFPath),
			types.MRP("python", pythonPath),
			types.MRP("esp_prov", espProvPath),
			types.MRP("command", pythonPath+" "+strings.Join(displayArgs, " ")),
			types.MRP("dry_run", true),
			types.MRP("read_passphrase_from_stdin", readPassphrase),
		))
	}

	runCtx, cancel := context.WithTimeout(ctx, time.Duration(s.Timeout)*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, pythonPath, args...)
	cmd.Env = append(os.Environ(), "IDF_PATH="+s.IDFPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	started := time.Now()
	err = cmd.Run()
	duration := time.Since(started)
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else if runCtx.Err() != nil {
			exitCode = -1
		} else {
			return err
		}
	}

	if addErr := gp.AddRow(ctx, types.NewRow(
		types.MRP("action", s.Action),
		types.MRP("service_name", s.ServiceName),
		types.MRP("ssid", s.SSID),
		types.MRP("pop", s.Pop),
		types.MRP("idf_path", s.IDFPath),
		types.MRP("python", pythonPath),
		types.MRP("esp_prov", espProvPath),
		types.MRP("command", pythonPath+" "+strings.Join(displayArgs, " ")),
		types.MRP("exit_code", exitCode),
		types.MRP("duration_ms", duration.Milliseconds()),
		types.MRP("read_passphrase_from_stdin", readPassphrase),
	)); addErr != nil {
		return addErr
	}

	if err != nil {
		if runCtx.Err() != nil {
			return fmt.Errorf("esp_prov.py timed out after %s", duration.Round(time.Second))
		}
		return fmt.Errorf("esp_prov.py failed with exit code %d", exitCode)
	}
	return nil
}

func buildBLEProvisionPythonArgs(s *BLEProvisionSettings, espProvPath string, passphrase string) ([]string, []string, error) {
	if s.Action == "version" {
		protoVer := s.ProtoVer
		if protoVer == "" {
			protoVer = "v1.1"
		}
		code := `import asyncio, os, sys
idf = os.environ['IDF_PATH']
sys.path.insert(0, idf + '/components/protocomm/python')
sys.path.insert(1, idf + '/tools/esp_prov')
import esp_prov
async def main():
    tp = await esp_prov.get_transport('ble', sys.argv[1])
    try:
        ok = await esp_prov.version_match(tp, sys.argv[2], True)
        print('==== Verified protocol version successfully ====' if ok else '==== Protocol version mismatch ====')
        raise SystemExit(0 if ok else 1)
    finally:
        await tp.disconnect()
asyncio.run(main())`
		args := []string{"-c", code, s.ServiceName, protoVer}
		displayArgs := []string{"-c", "<almanach-esp-prov-version-check>", s.ServiceName, protoVer}
		return args, displayArgs, nil
	}

	args := []string{espProvPath, "--transport", "ble", "--service_name", s.ServiceName, "--sec_ver", fmt.Sprint(s.SecVer)}
	if s.Pop != "" {
		args = append(args, "--pop", s.Pop)
	}
	if s.ProtoVer != "" {
		args = append(args, "--proto_ver", s.ProtoVer)
	}
	if s.Verbose {
		args = append(args, "--verbose")
	}

	switch s.Action {
	case "provision":
		args = append(args, "--ssid", s.SSID, "--passphrase", passphrase)
	case "reset":
		args = append(args, "--reset")
	case "reprov":
		args = append(args, "--reprov")
	default:
		return nil, nil, fmt.Errorf("unsupported action %q", s.Action)
	}

	return args, redactEspProvArgs(args), nil
}

func defaultIDFPath() string {
	if v := os.Getenv("IDF_PATH"); v != "" {
		return v
	}
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), "esp", "esp-idf-5.4.2"),
		filepath.Join(os.Getenv("HOME"), "esp", "esp-idf-5.4.1"),
		filepath.Join(os.Getenv("HOME"), "esp", "esp-idf"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
	}
	return ""
}

func defaultIDFPython() string {
	candidates := []string{
		filepath.Join(os.Getenv("HOME"), ".espressif", "python_env", "idf5.4_py3.13_env", "bin", "python"),
		filepath.Join(os.Getenv("HOME"), ".espressif", "python_env", "idf5.4_py3.12_env", "bin", "python"),
		filepath.Join(os.Getenv("HOME"), ".espressif", "python_env", "idf5.4_py3.11_env", "bin", "python"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
			return candidate
		}
	}
	return "python3"
}

func readLineFromStdin(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && len(line) == 0 {
		return "", fmt.Errorf("read passphrase from stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func redactEspProvArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i < len(out)-1; i++ {
		if out[i] == "--passphrase" {
			out[i+1] = "<redacted>"
		}
	}
	return out
}

func bleProvisionInstallHints(idfPath string) string {
	if idfPath == "" {
		idfPath = "$HOME/esp/esp-idf-5.4.2"
	}
	return fmt.Sprintf(`Linux BLE provisioning dependency hints:
  1. Ensure BlueZ is running and your user can access BLE adapters:
       systemctl status bluetooth
       bluetoothctl show
  2. Install ESP-IDF Python provisioning dependencies into the selected interpreter:
       cd %s
       ./install.sh --enable-pytest
     or, for the active ESP-IDF Python env:
       python -m pip install bleak protobuf cryptography
  3. Verify Espressif's tool directly:
       IDF_PATH=%s python %s/tools/esp_prov/esp_prov.py --transport ble --service_name ALM_0F2320 --sec_ver 1 --pop alm-0f2320 --ssid YOUR_WIFI --passphrase YOUR_PASSWORD
`, idfPath, idfPath, idfPath)
}
