package main

import (
	"testing"
	"net/http"
	"fmt"

	"io/ioutil"
	"log"
	"net/url"
	"bytes"
	"os"
	"mime/multipart"
	"path/filepath"
	"io"
)

func TestPump(t *testing.T) {

	response, err := http.Get("http://localhost:3008/thumb/asdf.jpg")
	if err != nil {
		t.Errorf("%v", response)
		return
	}
	defer response.Body.Close()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	responseString := string(responseData)

	fmt.Println(responseString) //{"assets/asdf.jpg":"assets/asdf.jpg is not existing file"}

}

func TestPumpPostBase64(t *testing.T){
	request_url := "http://localhost:3008/upload/base64"
	// 要 POST的 参数
	form := url.Values{
		"base64": {"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAUAAAAFCAYAAACNbyblAAAAHElEQVQI12P4//8/w38GIAXDIBKE0DHxgljNBAAO9TXL0Y4OHwAAAABJRU5ErkJggg=="},
	}
	// func Post(url string, bodyType string, body io.Reader) (resp *Response, err error) {
	//对form进行编码
	body := bytes.NewBufferString(form.Encode())
	rsp, err := http.Post(request_url, "application/x-www-form-urlencoded", body)
	if err != nil {
		panic(err)
	}
	defer rsp.Body.Close()
	body_byte, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(body_byte))
}

func TestPumpPostMultipart(t *testing.T){
	/*path1, _ := os.Getwd()
	path += "/img_2.jpg"*/
	extraParams := map[string]string{
		"title":       "A cat image",
	}
	request, err := newfileUploadRequest("http://localhost:3008/upload/multipart", extraParams, "files[]", "img_2.jpg")
	if err != nil {
		log.Fatal(err)
	}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Fatal(err)
	} else {
		body := &bytes.Buffer{}
		_, err := body.ReadFrom(resp.Body)
		if err != nil {
			log.Fatal(err)
		}
		resp.Body.Close()
		fmt.Println(resp.StatusCode)
		fmt.Println(resp.Header)
		fmt.Println(body.String())
	}
}


// Creates a new file upload http request with optional extra params
func newfileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

