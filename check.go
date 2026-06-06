package main; import ("fmt"; "proxy-core/internal/db"); func main() { db.InitDB("./data/proxy.db"); c, _ := db.GetGlobalConfig(); fmt.Println(c.WebPort) }
