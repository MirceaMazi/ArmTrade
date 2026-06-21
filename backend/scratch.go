package main

import (
	"encoding/json"
	"fmt"
	"armtrade-backend/services"
)

func main() {
	yahoo := services.NewYahooFinanceService()
	data, err := yahoo.GetInsiderData("AAPL")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	out, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(out))
}
