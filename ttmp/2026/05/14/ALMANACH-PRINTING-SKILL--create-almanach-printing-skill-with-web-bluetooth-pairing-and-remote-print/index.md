---
Title: Create Almanach Printing Skill with Web Bluetooth Pairing and Remote Print
Ticket: ALMANACH-PRINTING-SKILL
Status: active
Topics:
    - almanach
    - printing
    - web-bluetooth
    - ble
    - wifi-provisioning
    - remote-access
    - crib
DocType: index
Intent: long-term
Owners: []
RelatedFiles:
    - Path: ../../../../../../../../../.pi/agent/skills/almanach-printing/SKILL.md
      Note: The pi agent printing skill
    - Path: ../../../../../../../crib-k3s/gitops/kustomize/almanach/deployment.yaml
      Note: crib-k3s almanach deployment manifest
    - Path: ../../../../../../../crib-k3s/gitops/kustomize/almanach/ingress.yaml
      Note: crib-k3s almanach HTTPS ingress at almanach.crib.scapegoat.dev
    - Path: ../../../../../../../crib-k3s/gitops/kustomize/almanach/service.yaml
      Note: crib-k3s almanach service manifest
    - Path: README.render-service.md
      Note: Full service documentation
    - Path: internal/app/cmd_ble_provision.go
      Note: BLE provision CLI wrapping esp_prov.py and native Go client
    - Path: internal/app/cmd_print.go
      Note: Print command that renders and sends bitmap to printer
    - Path: internal/app/cmd_setup.go
      Note: Setup command serving BLE provisioning page on localhost
    - Path: internal/app/printer.go
      Note: Core sendBitmapToPrinter function for ESP32 /api/print/bitmap endpoint
    - Path: internal/app/server.go
      Note: HTTP server with /api/render
    - Path: internal/app/setup_device.go
      Note: Provisioned device store
    - Path: web/src/provisioning/ProvisioningWizard.jsx
      Note: React provisioning wizard UI
    - Path: web/src/provisioning/bluetooth-support.js
      Note: Web Bluetooth availability check
    - Path: web/src/provisioning/espidf-client.js
      Note: Web Bluetooth ESP-IDF provisioning client with Security 1
ExternalSources: []
Summary: ""
LastUpdated: 2026-05-14T14:27:18.875067943-04:00
WhatFor: ""
WhenToUse: ""
---



# Create Almanach Printing Skill with Web Bluetooth Pairing and Remote Print

## Overview

<!-- Provide a brief overview of the ticket, its goals, and current status -->

## Key Links

- **Related Files**: See frontmatter RelatedFiles field
- **External Sources**: See frontmatter ExternalSources field

## Status

Current status: **active**

## Topics

- almanach
- printing
- web-bluetooth
- ble
- wifi-provisioning
- remote-access
- crib

## Tasks

See [tasks.md](./tasks.md) for the current task list.

## Changelog

See [changelog.md](./changelog.md) for recent changes and decisions.

## Structure

- design/ - Architecture and design documents
- reference/ - Prompt packs, API contracts, context summaries
- playbooks/ - Command sequences and test procedures
- scripts/ - Temporary code and tooling
- various/ - Working notes and research
- archive/ - Deprecated or reference-only artifacts
