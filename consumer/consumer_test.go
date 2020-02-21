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

package consumer_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Shopify/sarama"
	"github.com/deckarep/golang-set"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestConsumerConstructorNoKafka(t *testing.T) {
	storageCfg := storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
	}
	storage, err := storage.New(storageCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer storage.Close()

	brokerCfg := broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
	consumer, err := consumer.New(brokerCfg, storage)
	if err == nil {
		t.Fatal("Error should be reported")
	}
	if consumer != nil {
		t.Fatal("consumer.New should return nil instead of Consumer implementation")
	}
}

func TestParseEmptyMessage(t *testing.T) {
	const message = ``
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "unexpected end of JSON input") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithWrongContent(t *testing.T) {
	const message = `{"this":"is", "not":"expected content"}`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for message that has improper content")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "Missing required attribute 'OrgID'") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithImproperJSON(t *testing.T) {
	const message = `"this_is_not_json_dude"`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for message that does not contain valid JSON")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "json: cannot unmarshal") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseProperMessage(t *testing.T) {
	const messageStr = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}"}
`
	message, err := consumer.ParseMessage([]byte(messageStr))
	if err != nil {
		t.Fatal(err)
	}
	if int(*message.Organization) != 1 {
		t.Fatal("OrgID is different", message.Organization)
	}
	if *message.ClusterName != "aaaaaaaa-bbbb-cccc-dddd-000000000000" {
		t.Fatal("Cluster name is different", *message.ClusterName)
	}
	if *message.Report != "{}" {
		t.Fatal("Report name is different", *message.Report)
	}
}

func TestParseProperMessageWrongClusterName(t *testing.T) {
	const message = `
{"OrgID":1,
 "ClusterName":"this is not a UUID",
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for a wrong ClusterName format")
	}
	errorMessage := err.Error()
	if !strings.HasPrefix(errorMessage, "Cluster name is not a UUID") {
		t.Fatal("Improper error message: " + errorMessage)
	}
}

func TestParseMessageWithoutOrgID(t *testing.T) {
	const message = `
{"ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func TestParseMessageWithoutClusterName(t *testing.T) {
	const message = `
{"OrgID":1,
 "Report":"{}"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func TestParseMessageWithoutReport(t *testing.T) {
	const message = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000"}
`
	_, err := consumer.ParseMessage([]byte(message))
	if err == nil {
		t.Fatal("Error is expected to be returned for empty message")
	}
}

func dummyConsumer(s storage.Storage, whitelist bool) consumer.Consumer {
	brokerCfg := broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
	if whitelist {
		brokerCfg.OrgWhitelist = mapset.NewSetWith(1)
	}
	return consumer.KafkaConsumer{
		Configuration:     brokerCfg,
		Consumer:          nil,
		PartitionConsumer: nil,
		Storage:           s,
	}
}
func TestProcessEmptyMessage(t *testing.T) {
	storage := helpers.MustGetMockStorage(t, true)
	defer storage.Close()

	c := dummyConsumer(storage, true)

	message := sarama.ConsumerMessage{}
	// messsage is empty -> nothing should be written into storage
	c.ProcessMessage(&message)
	cnt, err := storage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	if cnt != 0 {
		t.Fatal("ProcessMessage wrote anything into DB", cnt)
	}
}

func TestProcessCorrectMessage(t *testing.T) {
	storage := helpers.MustGetMockStorage(t, true)
	defer storage.Close()

	c := dummyConsumer(storage, true)

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020-01-23T16:15:59.478901889Z"}
`
	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	// messsage is empty -> nothing should be written into storage
	err := c.ProcessMessage(&message)
	if err != nil {
		t.Fatal(err)
	}
	cnt, err := storage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	if cnt == 0 {
		t.Fatal("ProcessMessage does not wrote anything into storage")
	}
	if cnt != 1 {
		t.Fatal("ProcessMessage does more writes than expected")
	}
}

func TestProcessingMessageWithClosedStorage(t *testing.T) {
	storage := helpers.MustGetMockStorage(t, true)

	c := dummyConsumer(storage, false)

	storage.Close()

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020-01-23T16:15:59.478901889Z"}
`

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	err := c.ProcessMessage(&message)
	if err == nil {
		t.Fatal(fmt.Errorf("Expected error because database was closed"))
	}
}

func TestProcessingMessageWithWrongDateFormat(t *testing.T) {
	storage := helpers.MustGetMockStorage(t, true)
	defer storage.Close()

	c := dummyConsumer(storage, true)

	const messageValue = `
{"OrgID":1,
 "ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
 "Report":"{}",
 "LastChecked":"2020.01.23 16:15:59"}
`

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageValue)
	err := c.ProcessMessage(&message)
	if _, ok := err.(*time.ParseError); err == nil || !ok {
		t.Fatal(fmt.Errorf(
			"Expected time.ParseError error because date format is wrong. Got %+v", err,
		))
	}
}

func TestNewConsumerWithMockedKafkaOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockedBroker := broker.MustNewMockBroker(t)
		defer mockedBroker.Close()

		_, err := consumer.New(mockedBroker.Configuration, helpers.MustGetMockStorage(t, true))
		if err != nil {
			t.Fatal(err)
		}
	}, 5*time.Second)
}

func TestNewConsumerWithMockedKafkaNonExistingTopicError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockedBroker := broker.MustNewMockBroker(t)
		defer mockedBroker.Close()

		_, err := consumer.New(broker.Configuration{
			Address:      mockedBroker.Configuration.Address,
			Topic:        "non existing topic",
			Group:        mockedBroker.Configuration.Group,
			Enabled:      mockedBroker.Configuration.Enabled,
			OrgWhitelist: mockedBroker.Configuration.OrgWhitelist,
		}, helpers.MustGetMockStorage(t, true))
		if err == nil {
			t.Fatal(fmt.Errorf("Error about wrong topic expected, got %v", err))
		}
	}, 5*time.Second)
}

func TestStartConsumerWithMockedKafkaOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockedBroker := broker.MustNewMockBroker(t)
		defer mockedBroker.Close()

		c, err := consumer.New(mockedBroker.Configuration, helpers.MustGetMockStorage(t, true))
		if err != nil {
			t.Fatal(err)
		}

		go func() {
			err = c.Start()
			if err != nil {
				panic(err)
			}
		}()

		err = c.Close()
		if err != nil {
			t.Fatal(err)
		}
	}, 5*time.Second)
}

func TestConsumerProcessMessageWithMockedKafkaOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockedBroker := broker.MustNewMockBroker(t)
		// defer mockedBroker.Close()

		c, err := consumer.New(mockedBroker.Configuration, helpers.MustGetMockStorage(t, true))
		if err != nil {
			t.Fatal(err)
		}

		go func() {
			err = c.Start()
			if err != nil {
				panic(err)
			}
		}()

		fmt.Println(mockedBroker.Configuration.Address)

		/*_, _, err = producer.ProduceMessage(mockedBroker.Configuration, `{
			"OrgID":1,
			"ClusterName":"aaaaaaaa-bbbb-cccc-dddd-000000000000",
			"Report":"{}"
		}`)*/
		_, _, err = producer.ProduceMessage(mockedBroker.Configuration, `asdfasdf`)
		if err != nil {
			t.Fatal(err)
		}

		time.Sleep(99999 * time.Second)

		return

		for {
			if c.GetNumberOfConsumedMessages() > 0 {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}

		err = c.Close()
		if err != nil {
			t.Fatal(err)
		}
	}, 100000*time.Second)
}
