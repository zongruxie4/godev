package test

import (
	"bufio"
	"log"
	"os"
	"strings"
)

func textExists(file_name string, text_search string) int {

	file, err := os.Open(file_name)
	if err != nil {
		log.Fatal(err)

	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, text_search) {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)

	}

	return count

}
