package cmd

import (
	"fmt"
	"github.com/coffeemakr/threema/gateway"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func init()  {
	rootCmd.AddCommand(sendImageCmd)
}

var sendImageCmd = &cobra.Command{
	Use: "send_image",
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetOutput(os.Stderr)

		to := args[0]
		from := args[1]
		secret := args[2]
		privateKey := args[3]
		imageFilePath := args[4]

		client, err := gateway.NewEncryptedClient(from, secret, privateKey)
		if err != nil {
			log.Fatalln(err)
		}
		messageID, err := client.SendImage(to, imageFilePath)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Println(messageID)
		return nil
	},
}