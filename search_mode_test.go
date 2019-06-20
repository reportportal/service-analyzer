/*
* Copyright 2019 EPAM Systems
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */
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
