package main

import (
	"fmt"
	"net/http"
)

func main() {
	loadBalancerEndpoint := "http://localhost:8001/data"
	for i := 0; i <= 10; i++ {
		resp, _ := http.Get(loadBalancerEndpoint)
		fmt.Println(i, resp.StatusCode)
	}
}
