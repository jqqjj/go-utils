package main

import (
	"io"
	"log"
	"net/http"

	"github.com/jqqjj/go-utils"
)

func main() {
	ja3 := "771,4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,35-13-65281-23-27-10-16-43-45-11-0-5-17613-65037-51-18-41,4588-29-23-24,0"
	t, _ := utils.NewTransport(ja3, "")
	c := http.Client{Transport: t}
	r, _ := http.NewRequest("GET", "https://tls.browserleaks.com/json", nil)
	resp, err := c.Do(r)
	if err != nil {
		log.Fatalln(err)
	}
	data, err := io.ReadAll(resp.Body)
	log.Fatalln(string(data))
}
