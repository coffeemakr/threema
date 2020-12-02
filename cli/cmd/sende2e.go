package cmd

import (
	"bufio"
	"fmt"
	"github.com/coffeemakr/threema/gateway"
	"io/ioutil"
	"log"
	"os"

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

		reader := bufio.NewReader(os.Stdin)
		message, err := ioutil.ReadAll(reader)
		if err != nil {
			log.Fatalln(err)
		}
		client, err := gateway.NewEncryptedClient(from, secret, privateKey)
		if err != nil {
			log.Fatalln(err)
		}
		messageID, err := client.SendTextMessage(to, string(message))
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(messageID)
	},
}
