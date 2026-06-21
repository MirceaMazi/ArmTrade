package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
	url := "https://query2.finance.yahoo.com/v10/finance/quoteSummary/AAPL?modules=netSharePurchaseActivity,insiderTransactions"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, _ := http.DefaultClient.Do(req)
	body, _ := io.ReadAll(resp.Body)
	var data map[string]interface{}
	json.Unmarshal(body, &data)
	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}
