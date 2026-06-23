// internal/services/currency.go
// Exchange rate fetching, caching, and currency conversion.
// MXN is always the source of truth.
// USD is computed at display time from cached rate.
// Rate refreshes every hour via background goroutine.

package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// exchangeRateAPIURL is the free endpoint for MXN→USD rate.
	exchangeRateAPIURL = "https://open.er-api.com/v6/latest/MXN"

	// refreshInterval is how often we refresh the cached rate.
	refreshInterval = 1 * time.Hour

	// fetchTimeout is the max time to wait for the API response.
	fetchTimeout = 10 * time.Second

	// fallbackMXNtoUSD is used if the API has never been reached.
	// Updated manually as a safety net. Approximate rate.
	fallbackMXNtoUSD = 0.058
)

// CurrencyService provides exchange rate caching and conversion.
type CurrencyService struct {
	db   *pgxpool.Pool
	mu   sync.RWMutex
	rate float64 // cached MXN→USD rate
}

// NewCurrencyService creates a CurrencyService and loads the initial rate.
func NewCurrencyService(db *pgxpool.Pool) *CurrencyService {
	cs := &CurrencyService{
		db:   db,
		rate: fallbackMXNtoUSD,
	}

	// Load from DB if available (last known good rate)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if cached, err := cs.loadFromDB(ctx); err == nil {
		cs.rate = cached
		slog.Info("currency rate loaded from cache", "mxn_to_usd", cached)
	}

	// Fetch fresh rate in background
	go cs.refresh()

	// Start hourly refresh loop
	go cs.startRefreshLoop()

	return cs
}

// Rate returns the current MXN→USD exchange rate.
func (cs *CurrencyService) Rate() float64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.rate
}

// ConvertMXNToUSD converts a MXN amount to USD using the cached rate.
func (cs *CurrencyService) ConvertMXNToUSD(mxn float64) float64 {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return math.Round(mxn*cs.rate*100) / 100
}

// FormatMXN formats a price as Mexican Pesos.
// Example: 1500000 → "$1,500,000 MXN"
func FormatMXN(amount float64) string {
	return fmt.Sprintf("$%s MXN", formatNumber(amount, 0))
}

// FormatUSD formats a price as US Dollars.
// Example: 85714.29 → "$85,714 USD"
func FormatUSD(amount float64) string {
	return fmt.Sprintf("$%s USD", formatNumber(amount, 0))
}

// formatNumber adds comma separators to a number.
func formatNumber(n float64, decimals int) string {
	if decimals == 0 {
		n = math.Round(n)
	}

	s := fmt.Sprintf("%.*f", decimals, n)

	// Split on decimal point if present
	parts := []byte(s)
	dotIdx := len(parts)
	for i, b := range parts {
		if b == '.' {
			dotIdx = i
			break
		}
	}

	// Add commas to integer part
	intPart := parts[:dotIdx]
	var result []byte
	for i, b := range intPart {
		if b == '-' {
			result = append(result, b)
			continue
		}
		remaining := len(intPart) - i
		if remaining > 0 && remaining%3 == 0 && i > 0 && intPart[i-1] != '-' {
			result = append(result, ',')
		}
		result = append(result, b)
	}

	// Append decimal part if present
	if dotIdx < len(parts) {
		result = append(result, parts[dotIdx:]...)
	}

	return string(result)
}

// ─── Internal ────────────────────────────────────────────────────────────────

// refresh fetches a fresh rate from the API and caches it.
func (cs *CurrencyService) refresh() {
	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
	defer cancel()

	rate, err := cs.fetchFromAPI(ctx)
	if err != nil {
		slog.Warn("failed to fetch exchange rate — using cached value",
			"error", err,
			"cached_rate", cs.Rate(),
		)
		return
	}

	cs.mu.Lock()
	cs.rate = rate
	cs.mu.Unlock()

	// Persist to DB
	if err := cs.saveToDB(ctx, rate); err != nil {
		slog.Warn("failed to cache exchange rate in DB", "error", err)
	}

	slog.Info("exchange rate refreshed", "mxn_to_usd", rate)
}

// startRefreshLoop refreshes the rate every hour.
func (cs *CurrencyService) startRefreshLoop() {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for range ticker.C {
		cs.refresh()
	}
}

// apiResponse represents the exchange rate API response.
type apiResponse struct {
	Result string             `json:"result"`
	Rates  map[string]float64 `json:"rates"`
}

// fetchFromAPI fetches the current MXN→USD rate from the public API.
func (cs *CurrencyService) fetchFromAPI(ctx context.Context) (float64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exchangeRateAPIURL, nil)
	if err != nil {
		return 0, fmt.Errorf("currency.fetchFromAPI: create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("currency.fetchFromAPI: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("currency.fetchFromAPI: status %d", resp.StatusCode)
	}

	var data apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("currency.fetchFromAPI: decode: %w", err)
	}

	if data.Result != "success" {
		return 0, fmt.Errorf("currency.fetchFromAPI: result=%s", data.Result)
	}

	rate, ok := data.Rates["USD"]
	if !ok {
		return 0, fmt.Errorf("currency.fetchFromAPI: USD rate not found")
	}

	if rate <= 0 {
		return 0, fmt.Errorf("currency.fetchFromAPI: invalid rate %f", rate)
	}

	return rate, nil
}

// saveToDB caches the rate in the exchange_rates table.
func (cs *CurrencyService) saveToDB(ctx context.Context, rate float64) error {
	query := `
		INSERT INTO exchange_rates (from_cur, to_cur, rate, source, fetched_at)
		VALUES ('MXN', 'USD', $1, 'exchangerate-api', NOW())
		ON CONFLICT (from_cur, to_cur)
		DO UPDATE SET rate = $1, fetched_at = NOW()`

	_, err := cs.db.Exec(ctx, query, rate)
	if err != nil {
		return fmt.Errorf("currency.saveToDB: %w", err)
	}

	return nil
}

// loadFromDB loads the last cached rate from the database.
func (cs *CurrencyService) loadFromDB(ctx context.Context) (float64, error) {
	query := `
		SELECT rate FROM exchange_rates
		WHERE from_cur = 'MXN' AND to_cur = 'USD'
		ORDER BY fetched_at DESC
		LIMIT 1`

	var rate float64
	err := cs.db.QueryRow(ctx, query).Scan(&rate)
	if err != nil {
		return 0, fmt.Errorf("currency.loadFromDB: %w", err)
	}

	return rate, nil
}
