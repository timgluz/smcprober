package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/grafana/grafana-foundation-sdk/go/cog"
	"github.com/grafana/grafana-foundation-sdk/go/common"
	"github.com/grafana/grafana-foundation-sdk/go/dashboard"
	"github.com/grafana/grafana-foundation-sdk/go/prometheus"
	"github.com/grafana/grafana-foundation-sdk/go/table"
)

type DashboardConfig struct {
	Name string
}

func main() {
	dashboardConfig := DashboardConfig{}
	flag.StringVar(&dashboardConfig.Name, "dashboard-name", "SmartCitizen device", "Name of the dashboard to create")
	flag.Parse()

	dashboardJSON, err := buildDashboard(dashboardConfig)
	if err != nil {
		fmt.Println("Error building dashboard:", err)
		os.Exit(1)
	}

	fmt.Println(string(dashboardJSON))
}

func buildDashboard(config DashboardConfig) ([]byte, error) {
	builder := dashboard.NewDashboardBuilder(config.Name).
		Uid("smartcitizen-device-dashboard").
		Tags([]string{"smartcitizen", "device", "sensors"}).
		Refresh("5m").
		Time("now-1h", "now").
		WithVariable(
			dashboard.NewQueryVariableBuilder("device").
				Label("name of selected device").
				Description("name of selected device").
				Query(dashboard.StringOrMap{
					String: cog.ToPtr("label_values(smartcitizen_device_info,name)"),
				}).
				Refresh(dashboard.VariableRefreshOnTimeRangeChanged).       // refresh=1 in JSON
				Sort(dashboard.VariableSortAlphabeticalCaseInsensitiveAsc). // sort=2 in JSON
				Multi(false),
		).
		WithPanel(
			table.NewPanelBuilder().
				Title("Device Details").
				Footer(
					common.NewTableFooterOptionsBuilder().
						EnablePagination(true).
						CountRows(false),
				).
				Height(6).
				Span(12).
				WithTarget(
					prometheus.NewDataqueryBuilder().
						Expr(`group by (name, uuid, description) (smartcitizen_device_info{name=~"$device"})`).
						Instant().
						Format(prometheus.PromQueryFormatTable).
						RefId("A"),
				),
		)

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
