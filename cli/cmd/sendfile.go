package cmd

import (
	"fmt"
	"os"

	"github.com/coffeemakr/threema/gateway"
	"github.com/spf13/cobra"
)

var (
	sendFileThumbnail string
)

func init() {
	sendFile.PersistentFlags().StringVar(&sendFileThumbnail, "i", "", "Thumbnail")
	rootCmd.AddCommand(sendFile)
}

var sendFile = &cobra.Command{
	Use: "send_file",
	Run: func(cmd *cobra.Command, args []string) {
		to := args[0]
		from := args[1]
		secret := args[2]
		privateKey := args[3]
		filePath := args[4]

		client, err := gateway.NewEncryptedClient(from, secret, privateKey)
		if err != nil {
			fail(err)
		}
		client.PublicKeyStore = gateway.NewInMemoryStore()

		file := &gateway.FilePath{
			Path:          filePath,
			ThumbnailPath: sendFileThumbnail,
		}
		messageID, err := client.SendFile(to, file, "")
		if err != nil {
			fail(err)
		}
		fmt.Println(messageID)
		return
	},
}

func fail(err error) {
	_, _ = fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
	os.Exit(1)
}
