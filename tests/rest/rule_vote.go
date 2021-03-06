/*
Copyright © 2020 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tests

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/verdverm/frisby"
)

const (
	improperClusterID = "000000000000000000000000000000000000"
	knownCluster      = "00000000-0000-0000-0000-000000000000"

	anyRule   = "0" // we don't care
	knownRule = "foo"

	unexpectedErrorStatusMessage = "Expected error status, but got '%s' instead"
)

var knownClustersForOrganization1 []string = []string{
	"00000000-0000-0000-0000-000000000000",
	"00000000-0000-0000-ffff-000000000000",
	"00000000-0000-0000-0000-ffffffffffff",
}

var unknownClusters []string = []string{
	"abcdef00-0000-0000-0000-000000000000",
	"abcdef00-0000-0000-1111-000000000000",
	"abcdef00-0000-0000-ffff-000000000000",
}

var improperClusterIDs []string = []string{
	"xyz",                                   // definitely wrong
	"00000000-0000-0000-0000-00000000000",   // shorter by one character
	"0000000-0000-0000-0000-000000000000",   // shorter by one character
	"00000000-000-0000-0000-000000000000",   // shorter by one character
	"00000000-0000-000-0000-000000000000",   // shorter by one character
	"00000000-0000-0000-000-000000000000",   // shorter by one character
	"00000000-0000-0000-0000-fffffffffffff", // longer by one character
	"00000000f-0000-0000-0000-ffffffffffff", // longer by one character
	"00000000-0000f-0000-0000-ffffffffffff", // longer by one character
	"00000000-0000-0000f-0000-ffffffffffff", // longer by one character
	"00000000-0000-0000-0000f-ffffffffffff", // longer by one character
}

var knownRules []string = []string{
	"foo",
	"bar",
	"xyzzy",
}

func constructURLVoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/like", apiURL, clusterID, ruleID)
}

func constructURLUnvoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/dislike", apiURL, clusterID, ruleID)
}

func constructURLResetVoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/reset_vote", apiURL, clusterID, ruleID)
}

func constructURLGetVoteForRule(clusterID string, ruleID string) string {
	return fmt.Sprintf("%sclusters/%s/rules/%s/get_vote/", apiURL, clusterID, ruleID)
}

func checkOkStatus(f *frisby.Frisby) {
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status != "ok" {
		f.AddError(fmt.Sprintf("Expected ok status, but got '%s' instead", statusResponse.Status))
	}
}

func checkOkStatusUserVote(url string, message string) {
	f := frisby.Create(message).Put(url)
	checkOkStatus(f)
}

func checkOkStatusGetVote(url string, message string) {
	f := frisby.Create(message).Get(url)
	checkOkStatus(f)
}

func checkInvalidUUIDFormat(f *frisby.Frisby) {
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status == "ok" {
		f.AddError(fmt.Sprintf(unexpectedErrorStatusMessage, statusResponse.Status))
	}
	if !strings.Contains(statusResponse.Status, "Error: 'invalid UUID ") {
		f.AddError(fmt.Sprintf("Unexpected error reported: %v", statusResponse.Status))
	}

	f.PrintReport()
}

func checkInvalidUUIDFormatGet(url string, message string) {
	f := frisby.Create(message).Get(url)
	checkInvalidUUIDFormat(f)
}

func checkInvalidUUIDFormatPut(url string, message string) {
	f := frisby.Create(message).Put(url)
	checkInvalidUUIDFormat(f)
}

func checkItemNotFound(f *frisby.Frisby) {
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	if statusResponse.Status == "ok" {
		f.AddError(fmt.Sprintf(unexpectedErrorStatusMessage, statusResponse.Status))
	}
	if !strings.Contains(statusResponse.Status, "was not found in the storage") {
		f.AddError(fmt.Sprintf("Unexpected error reported: %v", statusResponse.Status))
	}

	f.PrintReport()
}

func checkItemNotFoundGet(url string, message string) {
	f := frisby.Create(message).Get(url)
	checkItemNotFound(f)
}

func checkItemNotFoundPut(url string, message string) {
	f := frisby.Create(message).Put(url)
	checkItemNotFound(f)
}

// reproducerForIssue385 checks whether the issue https://github.com/RedHatInsights/insights-results-aggregator/issues/385 has been fixed
func reproducerForIssue385() {
	const message = "Reproducer for issue #385 (https://github.com/RedHatInsights/insights-results-aggregator/issues/385)"
	// vote
	url := constructURLVoteForRule(improperClusterID, anyRule)
	checkInvalidUUIDFormatPut(url, message)

	// unvote
	url = constructURLUnvoteForRule(improperClusterID, anyRule)
	checkInvalidUUIDFormatPut(url, message)

	// reset vote
	url = constructURLResetVoteForRule(improperClusterID, anyRule)
	checkInvalidUUIDFormatPut(url, message)
}

// test the specified rule vote/unvote/reset REST API endpoint by using selected checker function
func testRuleVoteAPIendpoint(clusters []string, rules []string, message string, urlConstructor func(string, string) string, checker func(string, string)) {
	for _, cluster := range clusters {
		for _, rule := range rules {
			url := urlConstructor(cluster, rule)
			checker(url, message)
		}
	}
}

// checkLikeKnownRuleForKnownCluster tests whether 'like' REST API endpoint works correctly for known rule and known cluster
func checkLikeKnownRuleForKnownCluster() {
	testRuleVoteAPIendpoint(knownClustersForOrganization1, knownRules,
		"Test whether 'like' REST API endpoint works correctly for known rule and known cluster",
		constructURLVoteForRule, checkOkStatusUserVote)
}

// checkDislikeKnownRuleForKnownCluster tests whether 'dislike' REST API endpoint works correctly for known rule and known cluster
func checkDislikeKnownRuleForKnownCluster() {
	testRuleVoteAPIendpoint(knownClustersForOrganization1, knownRules,
		"Test whether 'dislike' REST API endpoint works correctly for known rule and known cluster",
		constructURLUnvoteForRule, checkOkStatusUserVote)
}

// checkResetKnownRuleForKnownCluster tests whether 'reset vote' REST API endpoint works correctly for known rule and known cluster
func checkResetKnownRuleForKnownCluster() {
	testRuleVoteAPIendpoint(knownClustersForOrganization1, knownRules,
		"Test whether 'reset vote' REST API endpoint works correctly for known rule and known cluster",
		constructURLResetVoteForRule, checkOkStatusUserVote)
}

// checkLikeKnownRuleForUnknownCluster tests whether 'like' REST API endpoint works correctly for unknown rule and unknown cluster
func checkLikeKnownRuleForUnknownCluster() {
	testRuleVoteAPIendpoint(unknownClusters, knownRules,
		"Test whether 'like' REST API endpoint works correctly for known rule and unknown cluster",
		constructURLVoteForRule, checkItemNotFoundPut)
}

// checkDislikeKnownRuleForUnknownCluster tests whether 'dislike' REST API endpoint works correctly for unknown rule and unknown cluster
func checkDislikeKnownRuleForUnknownCluster() {
	testRuleVoteAPIendpoint(unknownClusters, knownRules,
		"Test whether 'dislike' REST API endpoint works correctly for known rule and unknown cluster",
		constructURLUnvoteForRule, checkItemNotFoundPut)
}

// checkResetKnownRuleForUnknownCluster tests whether 'reset vote' REST API endpoint works correctly for unknown rule and unknown cluster
func checkResetKnownRuleForUnknownCluster() {
	testRuleVoteAPIendpoint(unknownClusters, knownRules,
		"Test whether 'reset vote' REST API endpoint works correctly for known rule and unknown cluster",
		constructURLResetVoteForRule, checkItemNotFoundPut)
}

// checkLikeKnownRuleForImproperCluster tests whether 'like' REST API endpoint works correctly for known rule and improper cluster id
func checkLikeKnownRuleForImproperCluster() {
	testRuleVoteAPIendpoint(improperClusterIDs, knownRules,
		"Test whether 'like' REST API endpoint works correctly for known rule and improper cluster ID",
		constructURLVoteForRule, checkInvalidUUIDFormatPut)
}

// checkDislikeKnownRuleForImproperCluster tests whether 'dislike' REST API endpoint works correctly for known rule and improper cluster id
func checkDislikeKnownRuleForImproperCluster() {
	testRuleVoteAPIendpoint(improperClusterIDs, knownRules,
		"Test whether 'dilike' REST API endpoint works correctly for known rule and improper cluster ID",
		constructURLUnvoteForRule, checkInvalidUUIDFormatPut)
}

// checkResetKnownRuleForImproperCluster tests whether 'reset vote' REST API endpoint works correctly for known rule and improper cluster id
func checkResetKnownRuleForImproperCluster() {
	testRuleVoteAPIendpoint(improperClusterIDs, knownRules,
		"Test whether 'reset vote' REST API endpoint works correctly for known rule and improper cluster ID",
		constructURLResetVoteForRule, checkInvalidUUIDFormatPut)
}

// checkGetUserVoteForKnownCluster tests whether 'get_vote' REST API endpoint works correctly for known rule and known cluster
func checkGetUserVoteForKnownCluster() {
	testRuleVoteAPIendpoint(knownClustersForOrganization1, knownRules,
		"Test whether 'get_vote' REST API endpoint works correctly for known rule and known cluster",
		constructURLGetVoteForRule, checkOkStatusGetVote)
}

// checkGetUserVoteForUnknownCluster tests whether 'get_vote' REST API endpoint works correctly for known rule and unknown cluster
func checkGetUserVoteForUnknownCluster() {
	testRuleVoteAPIendpoint(unknownClusters, knownRules,
		"Test whether 'get_vote' REST API endpoint works correctly for known rule and unknown cluster",
		constructURLGetVoteForRule, checkItemNotFoundGet)
}

// checkGetUserVoteForImproperCluster tests whether 'get_vote' REST API endpoint works correctly for known rule and improper cluster
func checkGetUserVoteForImproperCluster() {
	testRuleVoteAPIendpoint(improperClusterIDs, knownRules,
		"Test whether 'get_vote' REST API endpoint works correctly for known rule and improper cluster",
		constructURLGetVoteForRule, checkInvalidUUIDFormatGet)
}

// RuleVoteResponse represents response containing rule votes
type RuleVoteResponse struct {
	RuleVote int    `json:"vote"`
	Status   string `json:"status"`
}

func voteForRule(cluster string, rule string) {
	url := constructURLVoteForRule(cluster, rule)
	checkOkStatusUserVote(url, "Let's vote")
}

func unvoteForRule(cluster string, rule string) {
	url := constructURLUnvoteForRule(cluster, rule)
	checkOkStatusUserVote(url, "Let's unvote")
}

func resetVoteForRule(cluster string, rule string) {
	url := constructURLResetVoteForRule(cluster, rule)
	checkOkStatusUserVote(url, "Let's reset voting")
}

func checkVoteForClusterAndRule(cluster string, rule string, expectedVote int) {
	url := constructURLGetVoteForRule(cluster, rule)

	f := frisby.Create("Read vote for rule").Get(url)
	r := RuleVoteResponse{}
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)

	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
		return
	}

	err = json.Unmarshal(text, &r)
	if err != nil {
		f.AddError(err.Error())
		return
	}

	if r.RuleVote != expectedVote {
		f.AddError(fmt.Sprintf("Expected vote: %d, actual: %d", expectedVote, r.RuleVote))
	}
}

func checkGetUserVoteAfterVote() {
	cluster := knownClustersForOrganization1[0]
	rule := knownRules[0]

	checkVoteForClusterAndRule(cluster, rule, 0)
	voteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 1)
	resetVoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 0)
}

func checkGetUserVoteAfterUnvote() {
	cluster := knownClustersForOrganization1[0]
	rule := knownRules[1]

	checkVoteForClusterAndRule(cluster, rule, 0)
	unvoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, -1)
	resetVoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 0)
}

func checkGetUserVoteAfterDoubleVote() {
	cluster := knownClustersForOrganization1[0]
	rule := knownRules[0]

	checkVoteForClusterAndRule(cluster, rule, 0)
	voteForRule(cluster, rule)
	voteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 1)
	resetVoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 0)
}

func checkGetUserVoteAfterDoubleUnvote() {
	cluster := knownClustersForOrganization1[0]
	rule := knownRules[1]

	checkVoteForClusterAndRule(cluster, rule, 0)
	unvoteForRule(cluster, rule)
	unvoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, -1)
	resetVoteForRule(cluster, rule)
	checkVoteForClusterAndRule(cluster, rule, 0)
}
