package confession

import (
	"io/ioutil"
	"log"
	"net/http"
)

const imgRecognitionAddress = "http://localhost:8080/"

type Client struct {
	httpClient *http.Client
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{},
	}
}

func (c *Client) Recognize(downloadResponse *http.Response) string {
	var msg string

	req, err := http.NewRequest("POST", imgRecognitionAddress, downloadResponse.Body)
	if err != nil {
		log.Println("error from server recognition", err)
		return msg
	}
	req.Header.Add("Content-Type", "image/png")

	// do request to server recognition.
	recognitionResponse, err := c.httpClient.Do(req)
	if err != nil {
		log.Println(err)
		return msg
	}
	defer func() {
		er := recognitionResponse.Body.Close()
		if er != nil {
			log.Println(er)
		}
	}()

	recognitionResponseBody, err := ioutil.ReadAll(recognitionResponse.Body)
	if err != nil {
		log.Println("error on read response from server recognition", err)
		return msg
	}
	msg = string(recognitionResponseBody)

	return msg
}