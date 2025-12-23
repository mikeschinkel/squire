package squire

import (
	"fmt"
)

func PrintSlice[T any](s []T) {
	for _, v := range s {
		fmt.Println(v)
	}
}
