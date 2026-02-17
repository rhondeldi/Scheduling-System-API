package Utils

import (
	"encoding/json"
	"fmt"
)

func PrettyPrint(data interface{}) {
	b, err := json.MarshalIndent(data, "", "    ")

	if err != nil {
		panic(err)
	}

	fmt.Print(string(b))
}
