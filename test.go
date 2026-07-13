package main

import (
    "encoding/json"
    "fmt"
)

func main() {
    msg := `{"id":null,"method":"mining.notify","params":["2236695","726febea637ba2cc416764d0823d36f4ce93ad420000a1880000000000000000","01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff5703049c0e1a2f706f6f6c696e2e636f6d2f66702f3936368e012d02f933eb25fabe6d6d48676e3d565bde7f33b2953fd6960638621987d82191d4835e4aa2978f0e476e1000000000000000","ffffffff0390a7b2120000000017a9141366dca425687186b9b54e27e4ed48163b5beb80870000000000000000266a24aa21a9ed16fcbfdaedcb5149a02b77e920c7112d5c0c5e7361017094e84224cc1fd918c300000000000000002b6a2952534b424c4f434b3aa32e074d1b58c5c3f1f2daf352a53e764fe6f983c4899815f7dba50d0089e6da00000000",["dae5e668b7f161dbfc7edb23ce4fbf4574e6650306c33751861f568570b5a64f"],"20000000","17021a42","6a511559",false]}`
    var m map[string]interface{}
    json.Unmarshal([]byte(msg), &m)
    
    isCleanJobs := false
    if params, ok := m["params"].([]interface{}); ok && len(params) > 0 {
        fmt.Println("Len params:", len(params))
        if len(params) >= 9 {
            if cj, ok := params[len(params)-1].(bool); ok {
                isCleanJobs = cj
                fmt.Println("bool cj:", cj)
            } else if cj, ok := params[len(params)-1].(string); ok && cj == "true" {
                isCleanJobs = true
            }
        }
    }
    fmt.Println("isCleanJobs:", isCleanJobs)
}
