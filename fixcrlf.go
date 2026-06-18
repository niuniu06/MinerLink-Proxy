package main

import (
	"bytes"
	"io/ioutil"
	"log"
)

func main() {
	data, err := ioutil.ReadFile("install.sh")
	if err != nil {
		log.Fatal(err)
	}
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
	err = ioutil.WriteFile("install.sh", data, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
