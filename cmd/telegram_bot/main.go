package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"./confession"
)

const (
	Token       = "telegram_token"
	GetUrl      = "api_telegram_bot"
	DownloadUrl = "api_telegram_file"
	MsgUrl      = "api_telegram_sendMessage"
)

type RequestWebhook struct {
	Message Msg
}

type Msg struct {
	MessageId int    `json:"message_id"`
	Text      string `json:"text"`
	From      struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"from"`
	Photo *[]PhotoSize `json:"photo"`
	Chat  struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		Username  string `json:"username"`
	} `json:"chat"`
	Date  int `json:"date"`
	Voice struct {
		Duration int64  `json:"duration"`
		MimeType string `json:"mime_type"`
		FileId   string `json:"file_id"`
		FileSize int64  `json:"file_size"`
	} `json:"voice"`
}

type PhotoSize struct {
	FileID   string `json:"file_id"`
	Width    int    `json:"width"`
	Height   int64  `json:"height"`
	FileSize int64  `json:"file_size"`
}

type ImgFileInfo struct {
	Ok     bool `json:"ok"`
	Result struct {
		FileId       string `json:"file_id"`
		FileUniqueId string `json:"file_unique_id"`
		FileSize     int    `json:"file_size"`
		FilePath     string `json:"file_path"`
	} `json:"result"`
}

type sendMessageReqBody struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	log.Println("Run BOT server: localhost:3000 ....")
	err := http.ListenAndServe(":3000", http.HandlerFunc(BotHandler))
	if err != nil {
		log.Fatalln(err)
	}
}

func BotHandler(w http.ResponseWriter, r *http.Request) {

	webhookBody := &RequestWebhook{}
	err := json.NewDecoder(r.Body).Decode(webhookBody)
	if err != nil {
		log.Println("could not decode request body", err)
		return
	}

	var downloadResponse *http.Response
	if webhookBody.Message.Photo == nil {
		log.Println("no photo in webhook body. webhookBody: ", webhookBody)
		return
	}
	for _, photoSize := range *webhookBody.Message.Photo {
		imgFileInfoUrl := fmt.Sprintf(GetUrl, Token, photoSize.FileID)
		rr, err := http.Get(imgFileInfoUrl)
		if err != nil {
			log.Println("unable retrieve img by FileID", err)
			return
		}
		defer rr.Body.Close()
		fileInfoJson, err := ioutil.ReadAll(rr.Body)
		if err != nil {
			log.Println("unable read img by FileID", err)
			return
		}
		imgInfo := &ImgFileInfo{}
		err = json.Unmarshal(fileInfoJson, imgInfo)
		if err != nil {
			log.Println("unable unmarshal file description from api.telegram by url: "+imgFileInfoUrl, err)
		}

		downloadFileUrl := fmt.Sprintf(DownloadUrl, Token, imgInfo.Result.FilePath)
		downloadResponse, err = http.Get(downloadFileUrl)
		if err != nil {
			log.Println("unable download file by file_path: "+downloadFileUrl, err)
			return
		}
		defer downloadResponse.Body.Close()
	}

	confessionCleint := confession.New()
	msg := confessionCleint.Recognize(downloadResponse)

	if err := sendResponseToUser(webhookBody.Message.Chat.ID, msg); err != nil {
		log.Println("error in sending reply: ", err)
		return
	}
}

func sendResponseToUser(chatID int64, msg string) error {

	msgBody := &sendMessageReqBody{
		ChatID: chatID,
		Text:   msg,
	}

	msgBytes, err := json.Marshal(msgBody)
	if err != nil {
		return err
	}

	res, err := http.Post(fmt.Sprintf(MsgUrl, Token), "application/json", bytes.NewBuffer(msgBytes))
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		buf := new(bytes.Buffer)
		_, err := buf.ReadFrom(res.Body)
		if err != nil {
			return err
		}
		return errors.New("unexpected status: " + res.Status)
	}

	return nil
}
