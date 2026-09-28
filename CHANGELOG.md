# Changelog

All notable changes to this project are documented in this file.

The `VERSION` file is the source of truth for the current release. Publish it by
pushing a git tag with the same value (for example `v0.0.2`); the pipeline refuses
to build a release whose tag does not match `VERSION`.

## unreleased

## v0.0.2

- added custom prometheus exporter
- added Grafana dashboard for device
- added alerting for device battery level
- added weather exporter with historical climate normals
- made the container registry configurable (Docker Hub by default)

## v0.0.1

- Initial release
- Added basic alerting cron job
