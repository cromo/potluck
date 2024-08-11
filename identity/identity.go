package identity

import (
	"log"

	"github.com/jaevor/go-nanoid"
)

var generator func() string

func GenerateID() string {
	if generator == nil {
		gen, err := nanoid.Canonic()
		if err != nil {
			log.Fatal(err)
		}
		generator = gen
	}
	return generator()
}
