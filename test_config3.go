package main

import (
	"encoding/json"
	"fmt"
	"proxy-core/internal/db"
)

func main() {
	db.InitDB("data.db")
	configs, _ := db.GetAllConfigs()
	b, _ := json.MarshalIndent(configs, "", "  ")
	fmt.Println(string(b))
}
