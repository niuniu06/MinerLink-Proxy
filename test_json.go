package main
import (
	"encoding/json"
	"fmt"
)
func main() {
	jobJSON := {"id":null,"method":"mining.notify","params":["2368734","14d9f9e454a623341caf0d29b6afe2fa3f096d91000022420000000000000000","01","ff",["88","d6"],"20","17","6a",false]}
	var msg map[string]interface{}
	json.Unmarshal([]byte(jobJSON), &msg)
	params := msg["params"].([]interface{})
	params[8] = true
	modBytes, _ := json.Marshal(msg)
	fmt.Println(string(modBytes))
}
