package main
import (
	"encoding/json"
	"fmt"
)
func forceCleanJobs(jobJSON string) string {
	var msg map[string]interface{}
	if err := json.Unmarshal([]byte(jobJSON), &msg); err == nil {
		if params, ok := msg["params"].([]interface{}); ok && len(params) > 8 {
			params[8] = true
			if modBytes, err := json.Marshal(msg); err == nil {
				return string(modBytes)
			}
		}
	}
	return jobJSON
}
func main() {
	jobJSON := {"id":null,"method":"mining.notify","params":["2368734","14d9f9e454a623341caf0d29b6afe2fa3f096d91000022420000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff57031e9e0e1a2f706f6f6c696e2e636f6d2f66702f3936378e01330277557eedfabe6d6debdc0044d1a76517d18ad184164abb8408d82e059956be5342cc8ca5dcad59611000000000000000","ffffffff032125c3120000000017a9141366dca425687186b9b54e27e4ed48163b5beb80870000000000000000266a24aa21a9ed583cb12cabed13d24984f77c87fa4fb1362a08e10c3000c4d9c7666e82ef26d100000000000000002b6a2952534b424c4f434b3aee596a34d7358ebe1f6cfb71e2a3a099b7d5582c47a99d3e3c58080e008a1a1200000000",["8887475cce99884b879da2f82f9ddebe08deb6a84563a267ad500314f871e90c","d64847a35c62b9fcdf866c52b16dffaa8628d5963743b8efc79189edd0f12187"],"20000000","1702369d","6a55fe6a",false]}
	fmt.Println(forceCleanJobs(jobJSON))
}
