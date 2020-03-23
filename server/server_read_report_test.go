// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{"non-int", testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param organization with value non-int. Error: unsigned integer expected"
		}`,
	})
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param organization with value -1. Error: unsigned integer expected"
		}`,
	})
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testBadClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "invalid cluster name format: 'aaaa'"}`,
	})
}

func TestReadNonExistingReport(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Body: fmt.Sprintf(
			`{"status":"Item with ID %v/%v was not found in the storage"}`, testdata.OrgID, testdata.ClusterName,
		),
	})
}

func TestHttpServer_readReportForCluster_NoContent(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"report": {
				"meta": {
					"count": 0,
					"last_checked_at": "1970-01-01T01:00:00+01:00"
				},
				"data":[]
			}
		}`,
	})
}

func TestReadReportDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"sql: database is closed"}`,
	})
}

func assertReportResponsesEqual(t *testing.T, expected, got string) {
	var expectedResponse, gotResponse struct {
		Status string               `json:"status"`
		Report types.ReportResponse `json:"report"`
	}

	err := helpers.JSONUnmarshalStrict([]byte(expected), &expectedResponse)
	helpers.FailOnError(t, err)
	err = helpers.JSONUnmarshalStrict([]byte(got), &gotResponse)
	helpers.FailOnError(t, err)

	assert.NotEmpty(
		t,
		expectedResponse.Status,
		"status is empty(probably json is completely wrong and unmarshal didn't do anything useful)",
	)
	assert.Equal(t, expectedResponse.Status, gotResponse.Status)
	assert.Equal(t, expectedResponse.Report.Meta, gotResponse.Report.Meta)
	// ignore the order
	assert.ElementsMatch(t, expectedResponse.Report.Rules, gotResponse.Report.Rules)
}

func TestReadReportWithContent(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// write some rule content into the DB
	err = dbStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report3RulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})
}
