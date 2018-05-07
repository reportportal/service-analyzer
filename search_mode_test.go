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
    "launchId": "5a0d84a8eff46f62cfd9cbe4",                   
    "launchName": "test-results",  
    "project": "analyzer",                       
    "testItems": []
  }
]`
		var launches []Launch
		err := json.Unmarshal([]byte(data), &launches)
		Expect(err).ShouldNot(HaveOccurred())
		Expect(launches[0].Conf.Mode).Should(BeEquivalentTo(SearchModeAll))
	})

	It("should serialize correctly from string correctly", func() {
		data := `[{"launchId":"","project":"","launchName":"name","analyzerConfig":{"isAutoAnalyzerEnabled":false,"analyzer_mode":"ALL","indexing_running":false}}]`
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
