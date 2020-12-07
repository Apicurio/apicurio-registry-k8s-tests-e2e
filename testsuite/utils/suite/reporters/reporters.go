package reporters

import (
	"encoding/json"
	"io/ioutil"
	"strconv"

	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/ginkgo/types"
)

type TextSummaryReporter struct {
	file string
}

type CIMessage struct {
	Text string `json:"text"`
}

var _ reporters.Reporter = &TextSummaryReporter{}

func NewTextSummaryReporter(file string) *TextSummaryReporter {
	return &TextSummaryReporter{
		file: file,
	}
}

func (reporter *TextSummaryReporter) SpecSuiteWillBegin(config config.GinkgoConfigType, summary *types.SuiteSummary) {

}

func (reporter *TextSummaryReporter) BeforeSuiteDidRun(setupSummary *types.SetupSummary) {

}

func (reporter *TextSummaryReporter) SpecWillRun(specSummary *types.SpecSummary) {

}

func (reporter *TextSummaryReporter) SpecDidComplete(specSummary *types.SpecSummary) {
}

func (reporter *TextSummaryReporter) AfterSuiteDidRun(setupSummary *types.SetupSummary) {
}

func (reporter *TextSummaryReporter) SpecSuiteDidEnd(summary *types.SuiteSummary) {

	result := "SUCCESS"
	if !summary.SuiteSucceeded {
		result = "FAILED"
	}

	content := "\n"
	content += summary.SuiteDescription + " executed. "

	content += "\n"
	countersResult := result + "! -- " +
		strconv.Itoa(summary.NumberOfTotalSpecs) + " Total | " +
		strconv.Itoa(summary.NumberOfPassedSpecs) + " Passed | " +
		strconv.Itoa(summary.NumberOfFailedSpecs) + " Failed | " +
		strconv.Itoa(summary.NumberOfPendingSpecs) + " Pending | " +
		strconv.Itoa(summary.NumberOfSkippedSpecs) + " Skipped"

	content += countersResult
	content += "\n"

	oldcontent, err := ioutil.ReadFile(reporter.file)
	if err != nil {
		panic(err)
	}
	obj := &CIMessage{}
	json.Unmarshal(oldcontent, obj)

	obj.Text = obj.Text + content

	newcontent, err := json.Marshal(obj)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(reporter.file, newcontent, 0644)
	if err != nil {
		panic(err)
	}

	// file, err := os.OpenFile(reporter.file, os.O_RDWR|os.O_APPEND, 0644)
	// if err != nil {
	// 	panic(err)
	// }

	// defer file.Close()

	// _, err = file.WriteString(content)

	// if err != nil {
	// 	panic(err)
	// }
}
