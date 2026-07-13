package main
import (
	"bytes"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
)
func main() {
	b, err := ioutil.ReadFile("log/btcby_capture.pcap")
	if err != nil { panic(err) }
	
	// Find sequences of characters that look like JSON objects
	re := regexp.MustCompile(` + "`" + `\{"([^"\\]|\\.)*"\s*:\s*.*?\}` + "`" + `)
	
	matches := re.FindAll(b, -1)
	for _, m := range matches {
		s := string(m)
		// Clean up null bytes if any
		s = strings.ReplaceAll(s, "\x00", "")
		if strings.Contains(s, "method") || strings.Contains(s, "result") || strings.Contains(s, "id") {
			fmt.Println(s)
		}
	}
}
