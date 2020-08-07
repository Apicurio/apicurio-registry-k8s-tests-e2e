package utils

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/gomega"
)

type Replacement struct {
	Old string
	New string
}

func Template(tempName string, templatePath string, replacings ...Replacement) *os.File {
	replacedFile, err := ioutil.TempFile("/tmp", tempName+"-*.yaml")
	Expect(err).ToNot(HaveOccurred())

	templateContent, err := ioutil.ReadFile(templatePath)
	Expect(err).ToNot(HaveOccurred())

	replacedStr := ""
	for _, rep := range replacings {
		replacedStr = strings.ReplaceAll(string(templateContent), rep.Old, rep.New)
	}

	err = ioutil.WriteFile(replacedFile.Name(), []byte(replacedStr), 0644)
	Expect(err).ToNot(HaveOccurred())

	return replacedFile
}
