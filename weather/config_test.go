package weather_test

import (
	"testing"

	"github.com/timgluz/smcprober/weather"
)

func TestConfig_ApplyDefaults_ClimateNormalWindowDays(t *testing.T) {
	cases := []struct {
		name  string
		input int
		want  int
	}{
		{"zero uses default", 0, weather.DefaultClimateNormalWindowDays},
		{"negative uses default", -5, weather.DefaultClimateNormalWindowDays},
		{"in-range value kept", 10, 10},
		{"at max kept", weather.MaxClimateNormalWindowDays, weather.MaxClimateNormalWindowDays},
		{"above max clamped", 1000, weather.MaxClimateNormalWindowDays},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := weather.Config{ClimateNormalWindowDays: tc.input}
			cfg.ApplyDefaults()
			if cfg.ClimateNormalWindowDays != tc.want {
				t.Errorf("ClimateNormalWindowDays = %d, want %d", cfg.ClimateNormalWindowDays, tc.want)
			}
		})
	}
}
