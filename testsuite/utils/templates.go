package utils

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/onsi/gomega"
)

//Replacement represents a text replacement
type Replacement struct {
	Old string
	New string
}

//Template creates a file under tmp directory, copies the contents of templatePath into that new file and applies all the replacements passed
func Template(tempName string, templatePath string, replacings ...Replacement) *os.File {
	replacedFile, err := ioutil.TempFile("/tmp", tempName+"-*.yaml")
	Expect(err).ToNot(HaveOccurred())

	templateContent, err := ioutil.ReadFile(templatePath)
	Expect(err).ToNot(HaveOccurred())

	replacedStr := ""
	for _, rep := range replacings {
		content := ""
		if replacedStr == "" {
			content = string(templateContent)
		} else {
			content = replacedStr
		}
		replacedStr = strings.ReplaceAll(content, rep.Old, rep.New)
	}

	err = ioutil.WriteFile(replacedFile.Name(), []byte(replacedStr), 0644)
	Expect(err).ToNot(HaveOccurred())

	return replacedFile
}
