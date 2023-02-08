package utils

import (
	"bufio"
	"fmt"
	"os"
)

func FileLineByLine(file string) ([]string, error) {
	readFile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)

	var s []string
	for fileScanner.Scan() {
		s = append(s, fileScanner.Text())
	}
	readFile.Close()

	return s, nil
}
