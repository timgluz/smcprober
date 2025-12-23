package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
)

const (
	DefaultConfigPath = "configs/device-dashboard.json"
	DefaultHeight     = 6
	MaxPanelHeight    = 20
	MaxPanelSpan      = 24
	DefaultSpan       = 8 // 24 / 3 columns
	DefaultChartType  = "gauge"
)

type SensorChartConfig struct {
	Title   string `json:"title"`
	Metric  string `json:"metric"`
	Panel   string `json:"panel"`
	Type    string `json:"type"`
	Query   string `json:"query"`
	Instant bool   `json:"instant,omitempty"`
	Span    uint32 `json:"span,omitempty"`
	Height  uint32 `json:"height,omitempty"`
}

type DashboardConfig struct {
	Title  string              `json:"title"`
	Charts []SensorChartConfig `json:"charts"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/device-dashboard.json", "Path to configuration file")
	flag.Parse()

	dashboardConfig, err := loadDashboardConfig(configPath)
	if err != nil {
		fmt.Println("Error loading dashboard config:", err)
		os.Exit(1)
	}

	if len(dashboardConfig.Charts) == 0 {
		fmt.Println("No charts defined in the dashboard config")
		os.Exit(1)
	}

	dashboardJSON, err := buildDashboard(dashboardConfig)
	if err != nil {
		fmt.Println("Error building dashboard:", err)
		os.Exit(1)
	}

	if dashboardJSON == nil {
		fmt.Println("Generated dashboard JSON is nil")
		os.Exit(1)
	}

	fmt.Println(string(dashboardJSON))
}

func buildDashboard(config *DashboardConfig) ([]byte, error) {
	if config == nil {
		return nil, fmt.Errorf("dashboard config is nil")
	}

	builder := dashboard.NewDashboardBuilder(config.Title).
		Uid("smartcitizen-device-details").
		Tags([]string{"smartcitizen", "device", "sensors"}).
		Refresh("5m").
		Time("now-1h", "now").
		Editable().
		WithVariable(
			dashboard.NewQueryVariableBuilder("device").
				Label("name of selected device").
				Description("name of selected device").
				Query(dashboard.StringOrMap{
					String: cog.ToPtr("label_values(smartcitizen_device_info,uuid)"),
				}).
				Refresh(dashboard.VariableRefreshOnTimeRangeChanged).       // refresh=1 in JSON
				Sort(dashboard.VariableSortAlphabeticalCaseInsensitiveAsc). // sort=2 in JSON
				Multi(false).
				IncludeAll(false),
		)

	var groupedCharts = make(map[string][]SensorChartConfig)
	for _, sensor := range config.Charts {
		groupedCharts[sensor.Panel] = append(groupedCharts[sensor.Panel], sensor)
	}

	// add device state panel first
	rowBuilder := dashboard.NewRowBuilder("Device Information")
	for _, chart := range groupedCharts["device"] {
		rowBuilder.WithPanel(newChartPanel(chart))
	}
	builder.WithRow(rowBuilder)
	delete(groupedCharts, "device")

	// add device sensor panels
	for panelName, charts := range groupedCharts {
		rowBuilder := dashboard.NewRowBuilder(panelName)

		for _, chart := range charts {
			rowBuilder.WithPanel(newChartPanel(chart))
		}

		builder.WithRow(rowBuilder)
	}

	dashboardObj, err := builder.Build()
	if err != nil {
		return nil, err
	}

	dashboardJSON, err := json.MarshalIndent(dashboardObj, "", "  ")
	if err != nil {
		return nil, err
	}

	return dashboardJSON, nil
}

func newChartPanel(config SensorChartConfig) *dashboard.PanelBuilder {
	queryBuilder := prometheus.NewDataqueryBuilder().
		Expr(config.Query).
		RefId("A")

	switch config.Type {
	case "table":
		queryBuilder.Format(prometheus.PromQueryFormatTable)
	default:
		queryBuilder.Format(prometheus.PromQueryFormatTimeSeries)
	}

	if config.Instant {
		queryBuilder.Instant()
	}

	var width = uint32(DefaultSpan)
	if config.Span > 0 && config.Span <= MaxPanelSpan {
		width = config.Span
	}

	var height = uint32(DefaultHeight)
	if config.Height > 0 && config.Height < MaxPanelHeight {
		height = config.Height
	}

	return dashboard.NewPanelBuilder().
		Title(config.Title).
		Type(config.Type).
		Height(height).
		Span(width).
		WithTarget(queryBuilder)
}

func loadDashboardConfig(path string) (*DashboardConfig, error) {
	cleanPath := filepath.Clean(os.ExpandEnv(path))
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to close dashboard config file: %v\n", closeErr)
		}
	}()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config DashboardConfig
	if err := json.Unmarshal(content, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
