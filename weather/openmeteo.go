package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrNoClimateData is returned when no historical daily observations fall
// within the configured day-of-year window for the requested date — e.g. a
// transient partial response, or a location Open-Meteo has no data for.
var ErrNoClimateData = fmt.Errorf("weather: no climate data available for this location/date")

// openMeteoDaily matches the "daily" object returned by Open-Meteo's
// historical archive API. Temperature fields use pointers because
// Open-Meteo returns JSON null for missing days, and encoding/json would
// otherwise silently decode null into 0.0 for a plain float64.
type openMeteoDaily struct {
	Time            []string   `json:"time"`
	TemperatureMax  []*float64 `json:"temperature_2m_max"`
	TemperatureMin  []*float64 `json:"temperature_2m_min"`
	TemperatureMean []*float64 `json:"temperature_2m_mean"`
}

type openMeteoArchiveResponse struct {
	Daily openMeteoDaily `json:"daily"`
}

func (p *HTTPProvider) GetClimateNormals(ctx context.Context, lat, lon float64, month time.Month, day int) (ClimateNormals, error) {
	endpoint, err := url.JoinPath(p.config.ClimateEndpoint, p.config.ClimateAPIPath)
	if err != nil {
		return ClimateNormals{}, fmt.Errorf("weather: failed to build URL: %w", err)
	}

	endDate := time.Now().UTC().AddDate(0, 0, -1) // yesterday — today's data isn't finalized yet
	startDate := endDate.AddDate(-p.config.ClimateNormalYears, 0, 0)

	params := url.Values{}
	params.Set("latitude", strconv.FormatFloat(lat, 'f', -1, 64))
	params.Set("longitude", strconv.FormatFloat(lon, 'f', -1, 64))
	params.Set("start_date", startDate.Format("2006-01-02"))
	params.Set("end_date", endDate.Format("2006-01-02"))
	params.Set("daily", "temperature_2m_max,temperature_2m_min,temperature_2m_mean")
	// Pin daily aggregation to UTC explicitly — Open-Meteo defaults to GMT day
	// boundaries when omitted, but leaving it implicit invites drift if that
	// default ever changes. All date matching below is UTC-anchored too.
	params.Set("timezone", "UTC")

	fullURL := endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return ClimateNormals{}, fmt.Errorf("weather: failed to create request: %w", err)
	}

	resp, err := p.climateClient.Do(req)
	if err != nil {
		return ClimateNormals{}, fmt.Errorf("weather: climate normals request failed: %w", err)
	}

	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil {
			p.logger.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return ClimateNormals{}, fmt.Errorf("weather: climate normals request failed with status %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return ClimateNormals{}, fmt.Errorf("weather: failed to read response body: %w", err)
	}

	var owm openMeteoArchiveResponse
	if err := json.Unmarshal(content, &owm); err != nil {
		return ClimateNormals{}, fmt.Errorf("weather: failed to parse response: %w", err)
	}

	daily := owm.Daily
	n := len(daily.Time)
	if len(daily.TemperatureMax) != n || len(daily.TemperatureMin) != n || len(daily.TemperatureMean) != n {
		return ClimateNormals{}, fmt.Errorf("weather: climate response has mismatched field lengths (time=%d, max=%d, min=%d, mean=%d)",
			n, len(daily.TemperatureMax), len(daily.TemperatureMin), len(daily.TemperatureMean))
	}

	windowDays := p.config.ClimateNormalWindowDays

	var recordMin, recordMax float64
	var sumMin, sumMax, sumMean float64
	matched := 0
	years := map[int]struct{}{}

	for i := 0; i < n; i++ {
		minPtr, maxPtr, meanPtr := daily.TemperatureMin[i], daily.TemperatureMax[i], daily.TemperatureMean[i]
		if minPtr == nil || maxPtr == nil || meanPtr == nil {
			continue // skip rows with any missing (null) field
		}

		rowDate, err := time.Parse("2006-01-02", daily.Time[i])
		if err != nil {
			return ClimateNormals{}, fmt.Errorf("weather: failed to parse date %q: %w", daily.Time[i], err)
		}

		if !withinDayOfYearWindow(rowDate, month, day, windowDays) {
			continue
		}

		if matched == 0 {
			recordMin, recordMax = *minPtr, *maxPtr
		} else {
			if *minPtr < recordMin {
				recordMin = *minPtr
			}
			if *maxPtr > recordMax {
				recordMax = *maxPtr
			}
		}
		sumMin += *minPtr
		sumMax += *maxPtr
		sumMean += *meanPtr
		matched++
		years[rowDate.Year()] = struct{}{}
	}

	if matched == 0 {
		return ClimateNormals{}, ErrNoClimateData
	}

	return ClimateNormals{
		Lat:             lat,
		Lon:             lon,
		Month:           month,
		Day:             day,
		TempRecordMin:   recordMin,
		TempRecordMax:   recordMax,
		TempAverageMin:  sumMin / float64(matched),
		TempAverageMax:  sumMax / float64(matched),
		TempAverageMean: sumMean / float64(matched),
		SampleYears:     len(years),
	}, nil
}

// leapReferenceYear is used to compute a year-independent day-of-year
// ordinal for both the target date and each row's date. 2000 is a leap
// year, so remapping any (month, day) — including Feb 29 — into it always
// produces a valid calendar date, unlike reconstructing the target date in
// an arbitrary row year (which silently rolls Feb 29 over to Mar 1 in a
// non-leap year, producing an asymmetric window).
const leapReferenceYear = 2000
const daysInLeapReferenceYear = 366

// withinDayOfYearWindow reports whether rowDate falls within windowDays of
// the target month/day, comparing day-of-year ordinals in a fixed leap
// reference year and taking the shorter distance around the year boundary
// (so windows near Dec 31 / Jan 1 wrap correctly). Remapping into a fixed
// leap year — rather than reconstructing the target date in rowDate's own
// year — avoids both leap-year drift and the Feb 29 normalization bug.
func withinDayOfYearWindow(rowDate time.Time, month time.Month, day, windowDays int) bool {
	target := time.Date(leapReferenceYear, month, day, 0, 0, 0, 0, time.UTC).YearDay()
	row := time.Date(leapReferenceYear, rowDate.Month(), rowDate.Day(), 0, 0, 0, 0, time.UTC).YearDay()

	diff := target - row
	if diff < 0 {
		diff = -diff
	}
	if wrapped := daysInLeapReferenceYear - diff; wrapped < diff {
		diff = wrapped
	}
	return diff <= windowDays
}
