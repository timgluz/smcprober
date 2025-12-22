package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
)

type SensorChartConfig struct {
	Title  string `json:"title"`
	Metric string `json:"metric"`
	Panel  string `json:"panel"`
	Type   string `json:"type"`
	Query  string `json:"query"`
}

type DashboardConfig struct {
	Title  string              `json:"title"`
	Charts []SensorChartConfig `json:"charts"`
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/device-dashboard.json", "Path to configuration file (not used currently)")
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
	rowBuilder.WithPanel(newDeviceInfoPanel())
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

	// Create the provisioning wrapper with folder
	provisioning := map[string]interface{}{
		"dashboard": dashboardObj,
		"folderId":  nil,
		"folderUid": "SmartCitizen", // This sets the folder
		"overwrite": true,
	}

	dashboardJSON, err := json.MarshalIndent(provisioning, "", "  ")
	if err != nil {
		return nil, err
	}

	return dashboardJSON, nil
}

func newDeviceInfoPanel() *dashboard.PanelBuilder {
	return dashboard.NewPanelBuilder().
		Title("Device Details").
		Type("table").
		Height(6).
		Span(8).
		WithTarget(
			prometheus.NewDataqueryBuilder().
				Expr(`group by (name, uuid, description) (smartcitizen_device_info{uuid=~"$device"})`).
				Instant().
				Format(prometheus.PromQueryFormatTable).
				RefId("A"),
		)
}

func newChartPanel(config SensorChartConfig) *dashboard.PanelBuilder {
	return dashboard.NewPanelBuilder().
		Title(config.Title).
		Type(config.Type).
		Height(6).
		Span(8).
		WithTarget(
			prometheus.NewDataqueryBuilder().
				Expr(config.Query).
				RefId("A").
				Format(prometheus.PromQueryFormatTimeSeries),
		)
}

func loadDashboardConfig(path string) (*DashboardConfig, error) {
	cleanPath := os.ExpandEnv(path)
	file, err := os.Open(cleanPath)
	if err != nil {
		return nil, err
	}

	defer file.Close()

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
