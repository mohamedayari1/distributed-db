package main
import (
	"fmt"
	"os"
)

func SaveData1(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0664)
	if err != nil {
		fmt.Println("| ta3mlo kachta")
	}

	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		fmt.Println("| ta3mlo kachta")
	}

	return file.Sync()
}

func main() {
	fmt.Println("hello distributed systems")
}