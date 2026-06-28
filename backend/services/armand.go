package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type ArmandService struct {
	yahooService *YahooFinanceService
}

func NewArmandService(yahoo *YahooFinanceService) *ArmandService {
	return &ArmandService{
		yahooService: yahoo,
	}
}

type ArmandAnalysisResponse struct {
	Recommendation  string   `json:"recommendation"`
	Reasoning       []string `json:"reasoning"`
	SocialSentiment string   `json:"socialSentiment"`
	Annotations     []struct {
		Date        string `json:"date"`
		Description string `json:"description"`
		Type        string `json:"type"` // e.g., "bullish", "bearish", "info"
	} `json:"annotations"`
}

type geminiRequest struct {
	Contents []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
	GenerationConfig struct {
		ResponseMimeType string `json:"responseMimeType"`
	} `json:"generationConfig"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

type AnalyzeRequest struct {
	Ticker  string `json:"ticker"`
	Persona string `json:"persona"`
	WhatIf  string `json:"whatIf"`
}

func (s *ArmandService) Analyze(req *AnalyzeRequest) (*ArmandAnalysisResponse, error) {
	ticker := req.Ticker
	// 1. Gather Context
	fundamentals, err := s.yahooService.GetFundamentals(ticker)
	if err != nil {
		return nil, fmt.Errorf("failed to get fundamentals for AI: %v", err)
	}

	chartData, err := s.yahooService.GetChart(ticker, "1d", "3mo")
	if err != nil {
		return nil, fmt.Errorf("failed to get chart data for AI: %v", err)
	}

	// 2. Build the Prompt
	prompt := s.buildPrompt(req, fundamentals, chartData)

	// 3. Call Gemini
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &ArmandAnalysisResponse{
			Recommendation: "HOLD",
			Reasoning: []string{
				"GEMINI_API_KEY environment variable is not set.",
				"Please add your Gemini API key to backend/.env",
				"Once added, Armand will analyze the stock automatically.",
			},
		}, nil
	}

	schema := `{
  "type": "object",
  "properties": {
    "recommendation": {"type": "string", "enum": ["BUY", "HOLD", "SELL"]},
    "reasoning": {"type": "array", "items": {"type": "string"}},
    "socialSentiment": {"type": "string", "enum": ["Bullish", "Bearish", "Neutral"]},
    "annotations": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "date": {"type": "string", "description": "YYYY-MM-DD format"},
          "description": {"type": "string"},
          "type": {"type": "string", "enum": ["bullish", "bearish", "info"]}
        },
        "required": ["date", "description", "type"]
      }
    }
  },
  "required": ["recommendation", "reasoning", "socialSentiment", "annotations"]
}`
	promptWithSchema := prompt + "\n\nAlso, identify the 2-3 most significant price movement days from the chart data and provide annotations for them. For each annotation:\n- Date in YYYY-MM-DD format\n- A description explaining the REAL-WORLD REASON for the price move (e.g., earnings report, contract win/loss, analyst upgrade/downgrade, geopolitical event, sector rotation, regulatory news). Use the recent news headlines provided above and your general knowledge of recent events involving this company to explain what caused each move. Do NOT just describe the price action itself (e.g., avoid 'stock dropped on high volume' or 'sharp break below support'). Instead explain the actual catalyst.\n- Whether it was bullish/bearish/info\n\nIMPORTANT: You must return the output strictly as a JSON object matching this schema:\n" + schema

	return callGeminiForType[ArmandAnalysisResponse](apiKey, promptWithSchema)
}

// Screen handles natural language stock screening via Gemini
type ScreenerResponse struct {
	Results []struct {
		Ticker string `json:"ticker"`
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"results"`
	Summary string `json:"summary"`
}

func (s *ArmandService) Screen(query string) (*ScreenerResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &ScreenerResponse{
			Summary: "GEMINI_API_KEY is not set. Please add your API key to backend/.env",
		}, nil
	}

	schema := `{
  "type": "object",
  "properties": {
    "results": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "ticker": {"type": "string"},
          "name": {"type": "string"},
          "reason": {"type": "string"}
        },
        "required": ["ticker", "name", "reason"]
      }
    },
    "summary": {"type": "string"}
  },
  "required": ["results", "summary"]
}`

	prompt := fmt.Sprintf(`You are Armand, an elite financial AI stock screener for the ArmTrade platform.
The user has asked the following natural language question:
"%s"

Identify 5-10 real, publicly traded stocks that match this criteria.
For each stock provide the ticker symbol, company name, and a one-sentence reason why it matches.
Also provide a brief summary of the screening results.

IMPORTANT: You must return the output strictly as a JSON object matching this schema:
%s`, query, schema)

	return callGeminiForType[ScreenerResponse](apiKey, prompt)
}

func (s *ArmandService) buildPrompt(req *AnalyzeRequest, fundamentals map[string]interface{}, chart map[string]interface{}) string {
	fBytes, _ := json.Marshal(fundamentals)
	cBytes, _ := json.Marshal(chart)

	personaStr := "Balanced Expert Analyst"
	if req.Persona != "" {
		personaStr = req.Persona
	}

	whatIfStr := ""
	if req.WhatIf != "" {
		whatIfStr = fmt.Sprintf("\nUser's 'What-If' Scenario / Question: \"%s\"\nProvide a risk assessment regarding this specific scenario.", req.WhatIf)
	}

	newsContext := ""
	// Fetch recent news so the AI can reference real events in its analysis
	if newsData, err := s.yahooService.GetNews(req.Ticker); err == nil && newsData != nil {
		if newsArray, ok := newsData["news"].([]interface{}); ok && len(newsArray) > 0 {
			newsContext = "\n\nRecent News Headlines for " + req.Ticker + ":\n"
			for i, item := range newsArray {
				if m, ok := item.(map[string]interface{}); ok {
					if title, ok := m["title"].(string); ok {
						newsContext += fmt.Sprintf("%d. %s\n", i+1, title)
					}
				}
			}
		}
	}

	return fmt.Sprintf(`You are Armand, an elite financial AI assistant for the ArmTrade platform.
Your current investment philosophy/persona is: **%s**.
Analyze the following stock data for %s and provide a clear investment recommendation (BUY, HOLD, or SELL).
Also, estimate the current Social Sentiment (Reddit/Twitter) based on recent general knowledge about this ticker (Bullish, Bearish, or Neutral).
%s

Provide 3-4 bullet points of reasoning based strictly on the data and your persona. 

Recent Fundamentals (JSON):
%s

Recent Price Action (3 Months) (JSON):
%s%s`, personaStr, req.Ticker, whatIfStr, string(fBytes), string(cBytes), newsContext)
}

type NewsItem struct {
	Title string `json:"title"`
	Link  string `json:"link"`
}

type NewsAnalysisResponse struct {
	Sentiment string `json:"sentiment"` // Bullish, Bearish, Neutral
	Reason    string `json:"reason"`
}

func (s *ArmandService) AnalyzeNewsSentiment(newsItems []NewsItem) ([]NewsAnalysisResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	defaultResponse := make([]NewsAnalysisResponse, len(newsItems))
	for i := range defaultResponse {
		defaultResponse[i] = NewsAnalysisResponse{Sentiment: "Neutral", Reason: "AI analysis requires API key"}
	}

	if apiKey == "" || len(newsItems) == 0 {
		return defaultResponse, nil
	}

	schema := `{
  "type": "array",
  "items": {
    "type": "object",
    "properties": {
      "sentiment": {"type": "string", "enum": ["Bullish", "Bearish", "Neutral"]},
      "reason": {"type": "string", "description": "1 sentence explanation"}
    },
    "required": ["sentiment", "reason"]
  }
}`

	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are a financial AI. Analyze the following news headlines and determine if they are Bullish, Bearish, or Neutral for the stock. Provide a 1-sentence reason for each.\n\n")
	for i, item := range newsItems {
		promptBuilder.WriteString(fmt.Sprintf("%d. %s\n", i+1, item.Title))
	}
	promptBuilder.WriteString(fmt.Sprintf("\nIMPORTANT: You must return the output strictly as a JSON array of objects matching this schema:\n%s", schema))

	result, err := callGeminiForType[[]NewsAnalysisResponse](apiKey, promptBuilder.String())
	if err != nil || result == nil || len(*result) != len(newsItems) {
		return defaultResponse, nil
	}

	return *result, nil
}

// CompareStocks performs an AI-driven side-by-side comparison of two stocks
type CompareResponse struct {
	Winner    string   `json:"winner"`
	Reasoning []string `json:"reasoning"`
	Ticker1   string   `json:"ticker1"`
	Ticker2   string   `json:"ticker2"`
	Summary1  string   `json:"summary1"`
	Summary2  string   `json:"summary2"`
	Verdict   string   `json:"verdict"`
}

func (s *ArmandService) CompareStocks(ticker1, ticker2 string) (*CompareResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &CompareResponse{
			Winner:    "N/A",
			Reasoning: []string{"GEMINI_API_KEY not set."},
			Ticker1:   ticker1,
			Ticker2:   ticker2,
		}, nil
	}

	fund1, _ := s.yahooService.GetFundamentals(ticker1)
	fund2, _ := s.yahooService.GetFundamentals(ticker2)
	f1Bytes, _ := json.Marshal(fund1)
	f2Bytes, _ := json.Marshal(fund2)

	schema := `{
  "type": "object",
  "properties": {
    "winner": {"type": "string"},
    "reasoning": {"type": "array", "items": {"type": "string"}},
    "ticker1": {"type": "string"},
    "ticker2": {"type": "string"},
    "summary1": {"type": "string"},
    "summary2": {"type": "string"},
    "verdict": {"type": "string"}
  },
  "required": ["winner", "reasoning", "ticker1", "ticker2", "summary1", "summary2", "verdict"]
}`

	prompt := fmt.Sprintf(`You are Armand, an elite financial AI for the ArmTrade platform.
Compare these two stocks and determine which is the better investment right now.

Stock 1: %s
Fundamentals: %s

Stock 2: %s
Fundamentals: %s

Provide:
1. "winner" - the ticker symbol of the better investment
2. "reasoning" - 4-5 bullet points explaining the comparison
3. "summary1" - a 2-sentence summary of stock 1's strengths/weaknesses
4. "summary2" - a 2-sentence summary of stock 2's strengths/weaknesses
5. "verdict" - a 1-sentence final investment verdict

IMPORTANT: Return strictly as JSON matching this schema:
%s`, ticker1, string(f1Bytes), ticker2, string(f2Bytes), schema)

	return callGeminiForType[CompareResponse](apiKey, prompt)
}

// SummarizeEarnings summarizes an earnings call transcript
type EarningsSummaryResponse struct {
	KeyPoints []string `json:"keyPoints"`
	Sentiment string   `json:"sentiment"`
	Outlook   string   `json:"outlook"`
	Risks     []string `json:"risks"`
}

func (s *ArmandService) SummarizeEarnings(transcript, ticker string) (*EarningsSummaryResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &EarningsSummaryResponse{
			KeyPoints: []string{"GEMINI_API_KEY not set."},
			Sentiment: "Neutral",
		}, nil
	}

	tickerContext := ""
	if ticker != "" {
		tickerContext = fmt.Sprintf("This is for ticker: %s.", ticker)
	}

	schema := `{
  "type": "object",
  "properties": {
    "keyPoints": {"type": "array", "items": {"type": "string"}, "description": "5 key takeaways"},
    "sentiment": {"type": "string", "enum": ["Bullish", "Bearish", "Neutral"]},
    "outlook": {"type": "string", "description": "1-2 sentence forward-looking outlook"},
    "risks": {"type": "array", "items": {"type": "string"}, "description": "2-3 key risks mentioned"}
  },
  "required": ["keyPoints", "sentiment", "outlook", "risks"]
}`

	prompt := fmt.Sprintf(`You are Armand, an elite financial AI for the ArmTrade platform.
%s
Summarize the following earnings call transcript. Extract:
1. "keyPoints" - exactly 5 key takeaways from the call
2. "sentiment" - overall sentiment (Bullish, Bearish, or Neutral)
3. "outlook" - 1-2 sentence forward-looking outlook
4. "risks" - 2-3 key risks or concerns mentioned

Transcript:
%s

IMPORTANT: Return strictly as JSON matching this schema:
%s`, tickerContext, transcript, schema)

	return callGeminiForType[EarningsSummaryResponse](apiKey, prompt)
}

// SectorSummaryRequest carries the context used to generate a sector overview.
type SectorSummaryRequest struct {
	Sector    string   `json:"sector"`
	Movers    []string `json:"movers"`
	Headlines []string `json:"headlines"`
}

// SectorSummaryResponse is the AI-generated prose overview for a sector.
type SectorSummaryResponse struct {
	Summary   string `json:"summary"`
	Sentiment string `json:"sentiment"`
}

// GenerateSectorSummary produces a short prose paragraph describing current
// conditions in a sector, derived from recent price movers and news headlines.
func (s *ArmandService) GenerateSectorSummary(req *SectorSummaryRequest) (*SectorSummaryResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &SectorSummaryResponse{
			Summary:   fmt.Sprintf("AI summary unavailable: GEMINI_API_KEY is not set. The %s sector view shows live prices and the latest headlines below.", req.Sector),
			Sentiment: "Neutral",
		}, nil
	}

	schema := `{
  "type": "object",
  "properties": {
    "summary": {"type": "string", "description": "2-4 sentence prose overview"},
    "sentiment": {"type": "string", "enum": ["Bullish", "Bearish", "Neutral"]}
  },
  "required": ["summary", "sentiment"]
}`

	var b strings.Builder
	b.WriteString(fmt.Sprintf("You are Armand, an elite financial AI for the ArmTrade platform.\nWrite a concise 2-4 sentence prose overview of current conditions in the %s sector.\nBase it strictly on the price movements and headlines provided. Do not invent figures.\n\n", req.Sector))
	if len(req.Movers) > 0 {
		b.WriteString("Recent price movements:\n")
		for _, m := range req.Movers {
			b.WriteString("- " + m + "\n")
		}
	}
	if len(req.Headlines) > 0 {
		b.WriteString("\nTop headlines:\n")
		for _, h := range req.Headlines {
			b.WriteString("- " + h + "\n")
		}
	}
	b.WriteString(fmt.Sprintf("\nIMPORTANT: Return strictly as JSON matching this schema:\n%s", schema))

	return callGeminiForType[SectorSummaryResponse](apiKey, b.String())
}

// --- Stock Relationship Network ---

type NetworkNode struct {
	Ticker      string `json:"ticker"`
	Name        string `json:"name"`
	Sector      string `json:"sector"`
	Description string `json:"description"`
}

type NetworkEdge struct {
	From         string `json:"from"`
	To           string `json:"to"`
	Relationship string `json:"relationship"`
	Label        string `json:"label"`
}

type NetworkResponse struct {
	CenterTicker string        `json:"centerTicker"`
	Nodes        []NetworkNode `json:"nodes"`
	Edges        []NetworkEdge `json:"edges"`
}

func (s *ArmandService) DiscoverNetwork(ticker string) (*NetworkResponse, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &NetworkResponse{
			CenterTicker: ticker,
			Nodes:        []NetworkNode{{Ticker: ticker, Name: ticker, Sector: "Unknown", Description: "GEMINI_API_KEY not set"}},
			Edges:        []NetworkEdge{},
		}, nil
	}

	// Optionally fetch fundamentals to give Gemini context about the company
	fundamentals, _ := s.yahooService.GetFundamentals(ticker)
	fundBytes, _ := json.Marshal(fundamentals)
	fundContext := ""
	if fundamentals != nil {
		fundContext = fmt.Sprintf("\n\nHere are the current fundamentals for %s to help you understand the company:\n%s", ticker, string(fundBytes))
	}

	schema := `{
  "type": "object",
  "properties": {
    "centerTicker": {"type": "string"},
    "nodes": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "ticker": {"type": "string", "description": "Stock ticker symbol"},
          "name": {"type": "string", "description": "Company name"},
          "sector": {"type": "string", "description": "Industry sector"},
          "description": {"type": "string", "description": "One sentence about what they do and how they relate"}
        },
        "required": ["ticker", "name", "sector", "description"]
      }
    },
    "edges": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "from": {"type": "string", "description": "Source ticker"},
          "to": {"type": "string", "description": "Target ticker"},
          "relationship": {"type": "string", "enum": ["supplier", "competitor", "customer", "partner", "subsidiary", "ripple"]},
          "label": {"type": "string", "description": "Short description of the relationship"}
        },
        "required": ["from", "to", "relationship", "label"]
      }
    }
  },
  "required": ["centerTicker", "nodes", "edges"]
}`

	prompt := fmt.Sprintf(`You are Armand, an elite financial AI assistant for the ArmTrade platform.

Given the stock ticker "%s", identify 20-30 publicly traded companies that are interconnected with it.
Think about the FULL supply chain, competitive ecosystem, AND adjacent domains:

**Direct relationships** (15-20 companies):
1. **Suppliers** - Who supplies key components, raw materials, or services to this company?
2. **Competitors** - Who directly competes in the same market segments?
3. **Customers** - Which major public companies are significant customers?
4. **Partners** - Who has strategic partnerships, joint ventures, or deep integrations?
5. **Subsidiaries** - Are there any publicly traded subsidiaries or parent companies?

**Ripple Effect / Adjacent Domains** (5-10 companies):
6. **Ripple** - Companies in ADJACENT industries that would be significantly affected by changes in %s's business.
   Think about the second-order and third-order effects:
   - If %s booms, which companies in OTHER sectors would benefit or suffer?
   - Example: NVDA (AI chips) → WDC (storage demand from AI data centers), SMCI (AI server assembly), VRT (data center cooling)
   - Example: TSLA (EVs) → ALB (lithium mining), ENPH (solar/energy ecosystem), CHPT (charging infrastructure)
   - These should be companies that an investor might NOT immediately think of but would be affected by the same macro trend.
   Use the relationship type "ripple" for these connections.

Also include connections BETWEEN the related companies (not just to the center), to create a rich network graph.
For example, if Company A supplies both the center company AND Company B, show that edge too.

Rules:
- Only include real, currently publicly traded companies with valid US ticker symbols.
- The center ticker "%s" must be included as a node.
- Aim for 20-30 nodes total (including the center) and 30-50 edges.
- Make the network feel interconnected, not just a star pattern. Include cross-connections between related companies.
- Each edge label should be concise (3-6 words) describing the specific relationship.
- Ripple-effect nodes should connect to existing nodes in the network where logical, not just to the center.
%s

IMPORTANT: Return strictly as JSON matching this schema:
%s`, ticker, ticker, ticker, ticker, fundContext, schema)

	return callGeminiForType[NetworkResponse](apiKey, prompt)
}

// --- Smart Money & Insider Radar ---

type InsiderPattern struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Sentiment   string `json:"sentiment"` // "bullish", "bearish", "neutral"
}

type KeyTransaction struct {
	InsiderName string  `json:"insiderName"`
	Title       string  `json:"title"`
	Type        string  `json:"type"` // "Buy", "Sell", "Option Exercise"
	Shares      int64   `json:"shares"`
	Value       float64 `json:"value"`
	Date        string  `json:"date"`
}

type InsiderAnalysis struct {
	SignalStrength  string           `json:"signalStrength"`  // "high", "moderate", "low", "neutral"
	Narrative       string           `json:"narrative"`       // AI summary
	Patterns        []InsiderPattern `json:"patterns"`
	KeyTransactions []KeyTransaction `json:"keyTransactions"`
	BuyVsSellRatio  string           `json:"buyVsSellRatio"`  // e.g., "3:1 Buy"
}

func (s *ArmandService) AnalyzeInsiderActivity(ticker string, insiderData map[string]interface{}) (*InsiderAnalysis, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return &InsiderAnalysis{
			SignalStrength: "neutral",
			Narrative:      "GEMINI_API_KEY not set — cannot analyze insider activity.",
			Patterns:       []InsiderPattern{},
			KeyTransactions: []KeyTransaction{},
			BuyVsSellRatio: "N/A",
		}, nil
	}

	// Serialize the raw Yahoo data so Gemini can analyze it
	dataBytes, _ := json.Marshal(insiderData)

	schema := `{
  "type": "object",
  "properties": {
    "signalStrength": {"type": "string", "enum": ["high", "moderate", "low", "neutral"]},
    "narrative": {"type": "string", "description": "2-4 sentence plain-English summary of what the insider activity suggests for investors"},
    "patterns": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "title": {"type": "string", "description": "Short pattern name, e.g. Cluster Buying Detected"},
          "description": {"type": "string", "description": "1-2 sentence explanation"},
          "sentiment": {"type": "string", "enum": ["bullish", "bearish", "neutral"]}
        },
        "required": ["title", "description", "sentiment"]
      }
    },
    "keyTransactions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "insiderName": {"type": "string"},
          "title": {"type": "string", "description": "Their corporate title e.g. CEO, CFO, Director"},
          "type": {"type": "string", "enum": ["Buy", "Sell", "Option Exercise"]},
          "shares": {"type": "integer"},
          "value": {"type": "number"},
          "date": {"type": "string", "description": "Date in YYYY-MM-DD format"}
        },
        "required": ["insiderName", "title", "type", "shares", "value", "date"]
      }
    },
    "buyVsSellRatio": {"type": "string", "description": "Ratio like 3:1 Buy or 1:5 Sell"}
  },
  "required": ["signalStrength", "narrative", "patterns", "keyTransactions", "buyVsSellRatio"]
}`

	prompt := fmt.Sprintf(`You are Armand, an elite financial AI assistant for the ArmTrade platform.

You have been given REAL insider trading data for %s sourced from SEC Form 4 filings via Yahoo Finance.
Your job is to ANALYZE this data and surface patterns — do NOT invent any transactions.

Here is the raw insider data:
%s

Analyze this data and identify:

1. **Signal Strength**: How significant is the insider activity?
   - "high" = Multiple insiders buying heavily, cluster buying, unusual large purchases
   - "moderate" = Some notable activity worth watching
   - "low" = Mostly routine, small transactions
   - "neutral" = No significant signal, or only option exercises / routine sales

2. **Narrative**: Write a 2-4 sentence plain-English summary of what the insider activity suggests.
   Be specific — mention names, amounts, and dates from the actual data.

3. **Patterns**: Identify 1-4 patterns (e.g., "Cluster Buying Before Earnings", "CEO Loading Up", "Director Selling After Run-Up").
   Only report patterns that are actually supported by the data.

4. **Key Transactions**: Extract the 3-8 most notable transactions from the data.
   These MUST be real transactions from the provided data — do not fabricate any.

5. **Buy vs Sell Ratio**: Summarize the overall direction (e.g., "3:1 Buy", "1:2 Sell", "Balanced").

IMPORTANT: Return strictly as JSON matching this schema:
%s`, ticker, string(dataBytes), schema)

	analysis, err := callGeminiForType[InsiderAnalysis](apiKey, prompt)
	if err == nil && analysis != nil {
		analysis.BuyVsSellRatio = extractBuySellRatio(insiderData)
	}
	return analysis, err
}

func extractBuySellRatio(data map[string]interface{}) string {
	qs, ok := data["quoteSummary"].(map[string]interface{})
	if !ok { return "N/A" }
	res, ok := qs["result"].([]interface{})
	if !ok || len(res) == 0 { return "N/A" }
	first, ok := res[0].(map[string]interface{})
	if !ok { return "N/A" }
	nspa, ok := first["netSharePurchaseActivity"].(map[string]interface{})
	if !ok { return "N/A" }

	buyCount := 0
	if bc, ok := nspa["buyInfoCount"].(map[string]interface{}); ok {
		if raw, ok := bc["raw"].(float64); ok {
			buyCount = int(raw)
		}
	}
	sellCount := 0
	if sc, ok := nspa["sellInfoCount"].(map[string]interface{}); ok {
		if raw, ok := sc["raw"].(float64); ok {
			sellCount = int(raw)
		}
	}

	if buyCount == 0 && sellCount == 0 {
		return "Neutral"
	}
	
	// Simplify the ratio
	gcd := func(a, b int) int {
		for b != 0 {
			t := b
			b = a % b
			a = t
		}
		return a
	}
	
	divisor := gcd(buyCount, sellCount)
	if divisor == 0 {
		divisor = 1
	}
	
	sBuy := buyCount / divisor
	sSell := sellCount / divisor
	
	if sBuy > sSell {
		return fmt.Sprintf("%d:%d Buy", sBuy, sSell)
	} else if sSell > sBuy {
		return fmt.Sprintf("%d:%d Sell", sBuy, sSell)
	}
	return "1:1 Neutral"
}

// callGeminiForType is a generic helper to call Gemini and parse the response into any type
func callGeminiForType[T any](apiKey, prompt string) (*T, error) {
	reqBody := geminiRequest{}
	reqBody.GenerationConfig.ResponseMimeType = "application/json"
	reqBody.Contents = []struct {
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}{
		{
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: prompt},
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	model := os.Getenv("GEMINI_MODEL")
	if model == "" {
		model = "gemini-2.5-flash"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gemini api error: %d, %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return nil, err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("empty response from Gemini")
	}

	jsonString := geminiResp.Candidates[0].Content.Parts[0].Text
	jsonString = strings.TrimPrefix(jsonString, "```json")
	jsonString = strings.TrimPrefix(jsonString, "```")
	jsonString = strings.TrimSuffix(jsonString, "```")
	jsonString = strings.TrimSpace(jsonString)

	var result T
	if err := json.Unmarshal([]byte(jsonString), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %v, raw: %s", err, jsonString)
	}

	return &result, nil
}
