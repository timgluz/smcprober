package weather_test

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/timgluz/smcprober/metric"
	"github.com/timgluz/smcprober/weather"
)

const (
	testClimateAPIPath = "v1/archive"
	testLocationName   = "Test"
)

// newClimateTestProvider spins up an httptest server returning fixture for
// any request, and returns a Provider configured against it. configure may
// be nil, or may mutate the Config before ApplyDefaults runs (e.g. to set a
// non-default window size).
func newClimateTestProvider(t *testing.T, fixture string, configure func(*weather.Config)) weather.Provider {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fixture))
	}))
	t.Cleanup(ts.Close)

	reg := metric.NewNamespacedRegistry(t.Name(), slog.Default())
	cfg := weather.Config{
		ClimateEndpoint: ts.URL,
		ClimateAPIPath:  testClimateAPIPath,
		Locations:       []weather.Location{{Name: testLocationName, Lat: 49.4811, Lon: 8.4353}},
	}
	if configure != nil {
		configure(&cfg)
	}
	cfg.ApplyDefaults()

	return weather.NewHTTPProvider(cfg, "unused", ts.Client(), reg, slog.Default())
}

// climateFixtureJSON spans three years (2022-2024). For a target of July 12
// with a +/-3 day window, each year contributes two in-window rows and one
// deliberately extreme out-of-window row that must be excluded from the
// aggregation. If the extreme rows leak in, the record/average assertions
// below will fail loudly.
const climateFixtureJSON = `{
    "daily": {
        "time": [
            "2022-07-12", "2022-07-13", "2022-07-20",
            "2023-07-12", "2023-07-11", "2023-07-25",
            "2024-07-12", "2024-07-14", "2024-07-01"
        ],
        "temperature_2m_min":  [10, 12, 999, 8, 14, -999, 11, 13, 500],
        "temperature_2m_max":  [20, 22, 999, 25, 24, -999, 28, 21, 500],
        "temperature_2m_mean": [15, 17, 999, 16, 19, -999, 19, 17, 500]
    }
}`

func TestHTTPProvider_GetClimateNormals(t *testing.T) {
	var gotQuery url.Values
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(climateFixtureJSON))
	}))
	defer ts.Close()

	reg := metric.NewNamespacedRegistry(t.Name(), slog.Default())
	cfg := weather.Config{
		ClimateEndpoint:         ts.URL,
		ClimateAPIPath:          testClimateAPIPath,
		ClimateNormalYears:      3,
		ClimateNormalWindowDays: 3,
		Locations:               []weather.Location{{Name: testLocationName, Lat: 49.4811, Lon: 8.4353}},
	}
	cfg.ApplyDefaults()
	provider := weather.NewHTTPProvider(cfg, "unused", ts.Client(), reg, slog.Default())

	cn, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.July, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotQuery.Get("latitude") == "" || gotQuery.Get("longitude") == "" {
		t.Error("missing latitude/longitude query params")
	}
	startDate, err := time.Parse("2006-01-02", gotQuery.Get("start_date"))
	if err != nil {
		t.Fatalf("start_date not a valid date: %v", err)
	}
	endDate, err := time.Parse("2006-01-02", gotQuery.Get("end_date"))
	if err != nil {
		t.Fatalf("end_date not a valid date: %v", err)
	}
	wantEnd := time.Now().AddDate(0, 0, -1)
	if endDate.Format("2006-01-02") != wantEnd.Format("2006-01-02") {
		t.Errorf("end_date = %s, want %s (yesterday)", endDate.Format("2006-01-02"), wantEnd.Format("2006-01-02"))
	}
	wantStart := wantEnd.AddDate(-cfg.ClimateNormalYears, 0, 0)
	if startDate.Format("2006-01-02") != wantStart.Format("2006-01-02") {
		t.Errorf("start_date = %s, want %s", startDate.Format("2006-01-02"), wantStart.Format("2006-01-02"))
	}

	const epsilon = 1e-9
	checkFloat := func(name string, got, want float64) {
		t.Helper()
		if math.Abs(got-want) > epsilon {
			t.Errorf("%s = %v, want %v", name, got, want)
		}
	}
	checkFloat("TempRecordMin", cn.TempRecordMin, 8)
	checkFloat("TempRecordMax", cn.TempRecordMax, 28)
	checkFloat("TempAverageMin", cn.TempAverageMin, (10.0+12.0+8.0+14.0+11.0+13.0)/6.0)
	checkFloat("TempAverageMax", cn.TempAverageMax, (20.0+22.0+25.0+24.0+28.0+21.0)/6.0)
	checkFloat("TempAverageMean", cn.TempAverageMean, (15.0+17.0+16.0+19.0+19.0+17.0)/6.0)
	if cn.SampleYears != 3 {
		t.Errorf("SampleYears = %d, want 3", cn.SampleYears)
	}
	if cn.Month != time.July || cn.Day != 12 {
		t.Errorf("Month/Day = %v/%d, want July/12", cn.Month, cn.Day)
	}
}

func TestHTTPProvider_GetClimateNormals_NonOKStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	reg := metric.NewNamespacedRegistry(t.Name(), slog.Default())
	cfg := weather.Config{
		ClimateEndpoint: ts.URL,
		ClimateAPIPath:  testClimateAPIPath,
		Locations:       []weather.Location{{Name: testLocationName, Lat: 49.4811, Lon: 8.4353}},
	}
	cfg.ApplyDefaults()
	provider := weather.NewHTTPProvider(cfg, "unused", ts.Client(), reg, slog.Default())

	_, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.July, 12)
	if err == nil {
		t.Fatal("expected error for non-200 status, got nil")
	}
}

func TestHTTPProvider_GetClimateNormals_WindowWrapsYearBoundary(t *testing.T) {
	// Target Jan 1 with a 3-day window should include Dec 30-31 of the
	// previous year as well as Jan 2-4 of the matching year.
	fixture := `{
        "daily": {
            "time": ["2023-12-30", "2023-06-15", "2024-01-02"],
            "temperature_2m_min":  [1, 999, 3],
            "temperature_2m_max":  [11, 999, 13],
            "temperature_2m_mean": [6, 999, 8]
        }
    }`
	provider := newClimateTestProvider(t, fixture, func(cfg *weather.Config) {
		cfg.ClimateNormalWindowDays = 3
	})

	cn, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.January, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cn.TempRecordMin != 1 || cn.TempRecordMax != 13 {
		t.Errorf("RecordMin/RecordMax = %v/%v, want 1/13 (the mid-June extreme row must be excluded)", cn.TempRecordMin, cn.TempRecordMax)
	}
	if cn.SampleYears != 2 {
		t.Errorf("SampleYears = %d, want 2 (2023-12-30 and 2024-01-02 are different calendar years)", cn.SampleYears)
	}
}

func TestHTTPProvider_GetClimateNormals_LeapYearDrift(t *testing.T) {
	// 2024 is a leap year, 2023 is not. A naive time.Time.YearDay() comparison
	// drifts by one day for dates after Feb 29 whenever the row's year and a
	// fixed reference year disagree on leap-year-ness. Target Nov 15 with a
	// 1-day window should match Nov 14-16 in BOTH years without drift.
	fixture := `{
        "daily": {
            "time": ["2023-11-15", "2023-11-17", "2024-11-15", "2024-11-17"],
            "temperature_2m_min":  [5, 999, 6, 999],
            "temperature_2m_max":  [15, 999, 16, 999],
            "temperature_2m_mean": [10, 999, 11, 999]
        }
    }`
	provider := newClimateTestProvider(t, fixture, func(cfg *weather.Config) {
		cfg.ClimateNormalWindowDays = 1
	})

	cn, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.November, 15)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cn.SampleYears != 2 {
		t.Errorf("SampleYears = %d, want 2 (Nov 17 rows at distance 2 must be excluded in both years)", cn.SampleYears)
	}
	if cn.TempRecordMax != 16 {
		t.Errorf("TempRecordMax = %v, want 16 (the distance-2 rows of 999 must be excluded)", cn.TempRecordMax)
	}
}

func TestHTTPProvider_GetClimateNormals_NoMatchingData(t *testing.T) {
	fixture := `{
        "daily": {
            "time": ["2023-01-01"],
            "temperature_2m_min":  [1],
            "temperature_2m_max":  [11],
            "temperature_2m_mean": [6]
        }
    }`
	provider := newClimateTestProvider(t, fixture, func(cfg *weather.Config) {
		cfg.ClimateNormalWindowDays = 1
	})

	cn, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.July, 12)
	if !errors.Is(err, weather.ErrNoClimateData) {
		t.Fatalf("expected ErrNoClimateData, got %v", err)
	}
	if (cn != weather.ClimateNormals{}) {
		t.Errorf("expected zero-value ClimateNormals on ErrNoClimateData, got %+v", cn)
	}
}

func TestHTTPProvider_GetClimateNormals_NullValues(t *testing.T) {
	fixture := `{
        "daily": {
            "time": ["2023-07-11", "2023-07-12", "2023-07-13"],
            "temperature_2m_min":  [5, null, 7],
            "temperature_2m_max":  [15, null, 17],
            "temperature_2m_mean": [10, null, 12]
        }
    }`
	provider := newClimateTestProvider(t, fixture, func(cfg *weather.Config) {
		cfg.ClimateNormalWindowDays = 3
	})

	cn, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.July, 12)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// The null row (July 12) must be excluded entirely, not treated as 0.0.
	if cn.TempRecordMin != 5 {
		t.Errorf("TempRecordMin = %v, want 5 (null row must not decode as 0)", cn.TempRecordMin)
	}
	if cn.SampleYears != 1 {
		t.Errorf("SampleYears = %d, want 1 (only July 11 and 13 are non-null, both 2023)", cn.SampleYears)
	}
}

func TestHTTPProvider_GetClimateNormals_MismatchedLength(t *testing.T) {
	fixture := `{
        "daily": {
            "time": ["2023-07-11", "2023-07-12"],
            "temperature_2m_min":  [5],
            "temperature_2m_max":  [15, 16],
            "temperature_2m_mean": [10, 11]
        }
    }`
	provider := newClimateTestProvider(t, fixture, nil)

	_, err := provider.GetClimateNormals(context.Background(), 49.4811, 8.4353, time.July, 12)
	if err == nil {
		t.Fatal("expected error for mismatched slice lengths, got nil (this must not panic)")
	}
}
