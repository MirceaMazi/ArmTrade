package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	neturl "net/url"
	"sync"
	"time"
)

const (
	headerUserAgent = "User-Agent"
	userAgentValue  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
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
	req1.Header.Set(headerUserAgent, userAgentValue)
	resp1, err := s.client.Do(req1)
	if err == nil {
		resp1.Body.Close()
	}

	// 2. Get crumb
	req2, _ := http.NewRequest("GET", "https://query1.finance.yahoo.com/v1/test/getcrumb", nil)
	req2.Header.Set(headerUserAgent, userAgentValue)
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

	req.Header.Set(headerUserAgent, userAgentValue)

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
		req2.Header.Set(headerUserAgent, userAgentValue)
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

// GetQuotes fetches batched quote data (price, change %, market cap, name) for the
// given comma-separated symbols. Uses the crumb-protected v7 quote endpoint with a
// single retry if the crumb has expired. Responses are cached for 5 minutes.
func (s *YahooFinanceService) GetQuotes(symbols string) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s&crumb=%s", symbols, s.crumb)

	res, err := s.cachedRequest(url, false, 5*time.Minute)
	if err != nil || res == nil || res["quoteResponse"] == nil {
		// Retry once with a refreshed crumb
		s.refreshCrumb()
		url = fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/quote?symbols=%s&crumb=%s", symbols, s.crumb)
		return s.cachedRequest(url, false, 5*time.Minute)
	}
	return res, nil
}

// SearchNews returns recent news items for an arbitrary query (e.g. a sector name).
// Cached for 5 minutes to respect Yahoo rate limits.
func (s *YahooFinanceService) SearchNews(query string, count int) (map[string]interface{}, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v1/finance/search?q=%s&newsCount=%d&quotesCount=0", neturl.QueryEscape(query), count)
	return s.cachedRequest(url, false, 5*time.Minute)
}

// GetEarningsCalendar fetches upcoming US earnings reports between the given dates
// (YYYY-MM-DD) using Yahoo's visualization calendar endpoint. Cached for 30 minutes.
func (s *YahooFinanceService) GetEarningsCalendar(from, to string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"sortType":     "ASC",
		"entityIdType": "earnings",
		"sortField":    "startdatetime",
		"includeFields": []string{
			"ticker", "companyshortname", "startdatetime", "startdatetimetype",
			"epsestimate", "epsactual", "epssurprisepct", "timeZoneShortName",
		},
		"query": map[string]interface{}{
			"operator": "and",
			"operands": []interface{}{
				map[string]interface{}{"operator": "gte", "operands": []interface{}{"startdatetime", from}},
				map[string]interface{}{"operator": "lte", "operands": []interface{}{"startdatetime", to}},
				map[string]interface{}{"operator": "eq", "operands": []interface{}{"region", "us"}},
			},
		},
		"offset": 0,
		"size":   100,
	}
	return s.calendarRequest(body)
}

// GetIPOCalendar fetches upcoming US IPOs between the given dates (YYYY-MM-DD) using
// Yahoo's visualization calendar endpoint. Cached for 30 minutes.
func (s *YahooFinanceService) GetIPOCalendar(from, to string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"sortType":     "ASC",
		"entityIdType": "ipo_info",
		"sortField":    "startdatetime",
		"includeFields": []string{
			"ticker", "companyshortname", "exchange", "offerprice",
			"dealtype", "startdatetime", "pricefrom", "priceto",
		},
		"query": map[string]interface{}{
			"operator": "and",
			"operands": []interface{}{
				map[string]interface{}{"operator": "gte", "operands": []interface{}{"startdatetime", from}},
				map[string]interface{}{"operator": "lte", "operands": []interface{}{"startdatetime", to}},
				map[string]interface{}{"operator": "eq", "operands": []interface{}{"region", "us"}},
			},
		},
		"offset": 0,
		"size":   100,
	}
	return s.calendarRequest(body)
}

// calendarRequest performs a cached POST against Yahoo's visualization endpoint,
// refreshing the crumb once if the first attempt fails.
func (s *YahooFinanceService) calendarRequest(body map[string]interface{}) (map[string]interface{}, error) {
	cacheKey, _ := json.Marshal(body)
	key := "calendar:" + string(cacheKey)

	s.mutex.Lock()
	if entry, ok := s.cache[key]; ok && time.Now().Before(entry.expiresAt) {
		s.mutex.Unlock()
		return entry.data, nil
	}
	s.mutex.Unlock()

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/visualization?crumb=%s", s.crumb)
	data, err := s.makePostRequest(url, body)
	if err != nil {
		s.refreshCrumb()
		url = fmt.Sprintf("https://query1.finance.yahoo.com/v1/finance/visualization?crumb=%s", s.crumb)
		data, err = s.makePostRequest(url, body)
		if err != nil {
			return nil, err
		}
	}

	s.mutex.Lock()
	s.cache[key] = cachedResponse{data: data, expiresAt: time.Now().Add(30 * time.Minute)}
	s.mutex.Unlock()
	return data, nil
}

func (s *YahooFinanceService) makePostRequest(url string, body map[string]interface{}) (map[string]interface{}, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set(headerUserAgent, userAgentValue)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("yahoo calendar API returned status: %d, body: %s", resp.StatusCode, string(respBody))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}
	return result, nil
}
