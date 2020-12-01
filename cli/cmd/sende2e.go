package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/coffeemakr/threema"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(e2eCmd)
}

var e2eCmd = &cobra.Command{
	Use:  "sende2e",
	Args: cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		log.SetOutput(os.Stderr)
		to := args[0]
		from := args[1]
		secret := args[2]
		privateKey := args[3]

		encryption, err := threema.ThreemaEncryption(privateKey)
		if err != nil {
			log.Fatalln(err)
		}
		reader := bufio.NewReader(os.Stdin)
		message, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatalln(err)
		}

		client := threema.GatewayClient{
			Secret: secret,
			ID:     from,
		}

		publicKey, err := client.LookupPublicKey(to)
		if err != nil {
			log.Fatalln(err)
		}

		encrypted, err := encryption.EncryptText(string(message[:]), publicKey)
		if err != nil {
			log.Fatalln(err)
		}

		messageID, err := client.SendMessage(to, encrypted)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(messageID)
	},
}
