package utils

import (
	"io"
	"net/http"
	"os"
	"strings"

	. "github.com/onsi/gomega"
)

func DownloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}

func ReaderToBytes(reader io.Reader) []byte {
	return []byte(ReaderToString(reader))
}

func ReaderToString(reader io.Reader) string {
	str := new(strings.Builder)
	_, err := io.Copy(str, reader)
	Expect(err).ToNot(HaveOccurred())
	return str.String()
}
