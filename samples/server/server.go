package main

import (
	"encoding/hex"
	"fmt"
	"github.com/coffeemakr/threema/gateway"
	"github.com/coffeemakr/threema/gateway/callback"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

func logError(format string, message ...interface{}) {
	log.Printf(fmt.Sprintf("ERROR - %s\n", format), message...)
}

func printMessages(client *gateway.EncryptedClient, values chan *callback.EncryptedMessage) {
	for callbackValue := range values {
		message, publicKey, err := client.DecryptMessage(callbackValue.From, callbackValue.Box, callbackValue.Nonce)
		if err != nil {
			logError("decryption failed: %s\n", err)
			continue
		}

		switch message.(type) {
		case *gateway.TextMessage:
			fmt.Printf("[%x] Text from %s (%s): %s\n",
				*callbackValue.MessageID,
				callbackValue.From,
				callbackValue.Nickname,
				message.(*gateway.TextMessage).Content)
		case *gateway.FileMessage:
			{
				fileMessage := message.(*gateway.FileMessage)
				fmt.Printf("[%x] File from %s (%s): %s\n",
					*callbackValue.MessageID,
					callbackValue.From,
					callbackValue.Nickname,
					fileMessage.FileName,
				)
				fileContent, err := client.DownloadFile(fileMessage.FileID, fileMessage.SharedKey)
				if err != nil {
					logError("download failed: %s", err)
					continue
				}
				filename := hex.EncodeToString(callbackValue.MessageID[:])
				if err = ioutil.WriteFile(filename, fileContent, 0600); err != nil {
					logError("download failed: %s", err)
					continue
				}
				fmt.Printf("[%x] File downloaded tp %s\n",
					*callbackValue.MessageID,
					filename)
			}
		case *gateway.ImageMessage: {
			{
				imageMessage := message.(*gateway.ImageMessage)
				fmt.Printf("[%x] Image from %s (%s)\n",
					*callbackValue.MessageID,
					callbackValue.From,
					callbackValue.Nickname,
				)
				fileContent, err := client.DownloadImage(imageMessage.BlobID, imageMessage.Nonce, publicKey)
				if err != nil {
					logError("download failed: %s", err)
					continue
				}
				filename := hex.EncodeToString(callbackValue.MessageID[:]) + ".jpg"
				if err = ioutil.WriteFile(filename, fileContent, 0600); err != nil {
					logError("download failed: %s", err)
					continue
				}
				fmt.Printf("[%x] Image downloaded to %s\n",
					*callbackValue.MessageID,
					filename)
			}
		}
		default:
			fmt.Printf("Other message received\n")
		}
	}
}

func main() {
	id := os.Args[1]
	apiSecret := os.Args[2]
	secretKey := os.Args[3]
	encryptedClient, err := gateway.NewEncryptedClient(id, apiSecret, secretKey)
	if err != nil {
		log.Fatalln(err)
	}
	values := make(chan *callback.EncryptedMessage, 100)
	go printMessages(encryptedClient, values)
	http.HandleFunc("/callback/", func(writer http.ResponseWriter, r *http.Request) {
		callbackValue, err := callback.ReadMessage(r, apiSecret)
		if err != nil {
			writer.WriteHeader(400)
			_, _ = writer.Write([]byte("thanks for nothing.\n"))
			return
		}
		select {
		case values <- callbackValue:
		default:
			log.Println("channel is full")
		}
		writer.WriteHeader(200)
		_, _ = writer.Write([]byte("thanks for all the fish!\n"))
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
