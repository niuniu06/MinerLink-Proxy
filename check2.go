package main; import ("flag"; "fmt"); func main() { p := flag.Int("api-port", 8080, ""); flag.Parse(); fmt.Println(*p) }
