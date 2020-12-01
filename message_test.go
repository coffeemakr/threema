package threema

import (
	"fmt"
	"testing"
)

func TestRandomPadding(t *testing.T) {
	for i := 0; i < 10000; i++ {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			got, err := RandomPadding()
			if err != nil {
				t.Fail()
			}
			if len(got) > 255 {
				t.Fail()
			}
			if len(got) == 0 {
				t.Fail()
			}
			if int(got[0]) != len(got) {
				t.Fail()
			}
		})
	}
}

func TestPackTextMessage(t *testing.T) {
	message, _ := PackTextMessage("hello threema")
	fmt.Printf("Message %x", message)
}
