package main
import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

func pearlSplitFunc(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	
	if data[0] == '{' || data[0] == '[' {
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			return i + 1, data[:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	if data[0] == '\n' || data[0] == '\r' {
		return 1, data[:1], nil
	}

	return 1, data[:1], nil
}

func main() {
	input := "{\"id\":1}\n"
	scanner := bufio.NewScanner(strings.NewReader(input))
	scanner.Split(pearlSplitFunc)
	for scanner.Scan() {
		fmt.Printf("TOKEN: %s\n", string(scanner.Bytes()))
	}
	if err := scanner.Err(); err != nil {
		fmt.Printf("ERROR: %v\n", err)
	}
}
