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

	chartData, err := s.yahooService.GetChart(ticker, "1d", "1mo")
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
	promptWithSchema := prompt + "\n\nAlso, identify the 2-3 most significant price movement days from the chart data and provide annotations for them (date in YYYY-MM-DD, a brief description of why the move happened, and whether it was bullish/bearish/info).\n\nIMPORTANT: You must return the output strictly as a JSON object matching this schema:\n" + schema

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

	return fmt.Sprintf(`You are Armand, an elite financial AI assistant for the ArmTrade platform.
Your current investment philosophy/persona is: **%s**.
Analyze the following stock data for %s and provide a clear investment recommendation (BUY, HOLD, or SELL).
Also, estimate the current Social Sentiment (Reddit/Twitter) based on recent general knowledge about this ticker (Bullish, Bearish, or Neutral).
%s

Provide 3-4 bullet points of reasoning based strictly on the data and your persona. 

Recent Fundamentals (JSON):
%s

Recent Price Action (1 Month) (JSON):
%s`, personaStr, req.Ticker, whatIfStr, string(fBytes), string(cBytes))
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
