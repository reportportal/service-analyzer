package main

import (
	"encoding/json"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"testing"
)

func TestSearchType(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Analyzer Tests")
}

var _ = Describe("SearchType", func() {
	It("should return correct string()", func() {
		Expect(SearchModeAll.String()).To(BeEquivalentTo("ALL"))
	})
	It("should parse from string correctly", func() {
		Expect(FromString("LAUNCH_NAME")).To(BeEquivalentTo(SearchModeLaunchName))
	})

	It("should deserialize correctly from string correctly", func() {
		data := `[
  {
    "analyzeMode": "ALL",                   
    "launchId": 1,                   
    "launchName": "test-results",  
    "project": 12,                       
    "testItems": []
  }
]`
		var launches []Launch
		err := json.Unmarshal([]byte(data), &launches)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(launches[0].Conf.Mode).Should(BeEquivalentTo(SearchModeAll))
	})

	It("should serialize correctly from string correctly", func() {
		data := `[{"launchId":0,"project":0,"launchName":"name","analyzerConfig":{"isAutoAnalyzerEnabled":false,"analyzerMode":"ALL","indexingRunning":false}}]`
		launches := []Launch{{
			Conf: AnalyzerConf{
				Mode: SearchModeAll,
			},
			LaunchName: "name",
		}}
		d, err := json.Marshal(launches)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(string(d)).Should(BeEquivalentTo(data))
	})
})
