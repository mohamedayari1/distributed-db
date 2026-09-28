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
		fmt.Println("| error while writing")
	}

	return file.Sync()
}


func SaveData2(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.tmp.%d", path, randomInt())
	fp, err:= os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0664)
	if err != nil {
		return err
	}

	defer func() {
		fp.Close()
		if err != nil {
			os.Remove(tmp)
		}
	}()
	_, err = fp.Write(data)
	if err != nil {
		return err
	}
	return os.Rename(tmp, path)
}