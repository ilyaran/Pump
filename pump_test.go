package main

import (
	"testing"
	"net/http"
	"fmt"

)

func TestPump(t *testing.T) {

	response, err := http.Get("http://localhost:3008/thumb/asdf.jpg")
	if err != nil {
		t.Errorf("%v", response)
		return
	}
	fmt.Print(response)

}