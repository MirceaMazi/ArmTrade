package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"sync"
	"time"
)

type cachedResponse struct {
	data      map[string]interface{}
	expiresAt time.Time
}

type YahooFinanceService struct {
	client *http.Client
	crumb  string
	mutex  sync.Mutex
	cache  map[string]cachedResponse
}

func NewYahooFinanceService() *YahooFinanceService {
	jar, _ := cookiejar.New(nil)
	svc := &YahooFinanceService{
		client: &http.Client{
			Timeout: 10 * time.Second,
			Jar:     jar,
		},
		cache: make(map[string]cachedResponse),
	}
	svc.refreshCrumb()
	return svc
}

func (s *YahooFinanceService) refreshCrumb() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 1. Get cookies
	req1, _ := http.NewRequest("GET", "https://fc.yahoo.com", nil)
	req1.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	resp1, err := s.client.Do(req1)
	if err == nil {
		resp1.Body.Close()
	}

	// 2. Get crumb
	req2, _ := http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
	resp2, err := s.client.Do(req2)
	if err != nil {
		return err
	}
	defer resp2.Body.Close()

	if resp2.StatusCode == 200 {
		crumbBytes, _ := io.ReadAll(resp2.Body)
		s.crumb = string(crumbBytes)
	}
	return nil
}

func (s *YahooFinanceService) Search(query string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v1/finance/search?q=%s&quotesCount=10", query)
	return s.makeRequest(url, false)
}

func (s *YahooFinanceService) GetNews(ticker string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v1/finance/search?q=%s&newsCount=3", ticker)
	return s.makeRequest(url, false)
}

func (s *YahooFinanceService) GetDividends(ticker string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=10y&events=div", ticker)
	return s.makeRequest(url, false)
}

func (s *YahooFinanceService) GetChart(ticker, interval, trange string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s", ticker, interval, trange)
	// Cache real-time price calls (1d/5d) for 60s to avoid hammering Yahoo
	ttl := 5 * time.Minute
	if trange == "5d" || trange == "1d" {
		ttl = 60 * time.Second
	}
	res, err := s.cachedRequest(url, false, ttl)
	if err != nil {
		// Retry with refreshed crumb once
		s.refreshCrumb()
		return s.cachedRequest(url, false, ttl)
	}
	return res, nil
}

func (s *YahooFinanceService) GetFundamentals(ticker string) (map[string]interface{}, error) {
	modules := "financialData,defaultKeyStatistics,assetProfile,summaryDetail,earnings"
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s&crumb=%s", ticker, modules, s.crumb)

	res, err := s.cachedRequest(url, true, 5*time.Minute)
	if err != nil || res == nil || res["quoteSummary"] == nil {
		// Retry once if crumb expired
		s.refreshCrumb()
		url = fmt.Sprintf("https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=%s&crumb=%s", ticker, modules, s.crumb)
		return s.cachedRequest(url, true, 5*time.Minute)
	}
	return res, nil
}

func (s *YahooFinanceService) cachedRequest(url string, useCrumb bool, ttl time.Duration) (map[string]interface{}, error) {
	s.mutex.Lock()
	if entry, ok := s.cache[url]; ok && time.Now().Before(entry.expiresAt) {
		s.mutex.Unlock()
		return entry.data, nil
	}
	s.mutex.Unlock()

	data, err := s.makeRequest(url, useCrumb)
	if err != nil {
		return nil, err
	}

	s.mutex.Lock()
	s.cache[url] = cachedResponse{data: data, expiresAt: time.Now().Add(ttl)}
	s.mutex.Unlock()
	return data, nil
}

func (s *YahooFinanceService) makeRequest(url string, useCrumb bool) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// On rate limit, wait briefly and retry once
	if resp.StatusCode == http.StatusTooManyRequests {
		resp.Body.Close()
		time.Sleep(2 * time.Second)
		req2, _ := http.NewRequest("GET", url, nil)
		req2.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
		resp, err = s.client.Do(req2)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("yahoo API returned status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// MakeRawRequest exposes makeRequest for other packages (e.g., market handlers)
func (s *YahooFinanceService) MakeRawRequest(url string) (map[string]interface{}, error) {
	return s.makeRequest(url, false)
}
