package broker

import (
	"context"
	"testing"

	"github.com/travisjeffery/jocko/jocko"
	"github.com/travisjeffery/jocko/jocko/config"
	"github.com/travisjeffery/jocko/protocol"
)

// MockBroker mock broker which works like Kafka
// see https://github.com/travisjeffery/jocko
type MockBroker struct {
	Configuration Configuration
	Connection    *jocko.Conn
	broker        *jocko.Server
}

const (
	defaultTopicName          = "topic"
	defaultNumberOfPartitions = 8
)

// MustNewMockBroker - initializes mock broker and returns you an instance
// calls t.Fatal on error
func MustNewMockBroker(t *testing.T) *MockBroker {
	mockBroker, err := NewMockBroker(t, defaultTopicName, defaultNumberOfPartitions)
	if err != nil {
		t.Fatal(err)
	}

	return mockBroker
}

// NewMockBroker - initializes mock broker and returns you an instance
func NewMockBroker(t *testing.T, topic string, numOfPartitions int32) (*MockBroker, error) {
	c, _ := jocko.NewTestServer(&testing.T{}, func(cfg *config.Config) {
		cfg.Bootstrap = true
		cfg.BootstrapExpect = 1
		cfg.StartAsLeader = true
	}, nil)
	if err := c.Start(context.Background()); err != nil {
		return nil, err
	}

	conn, err := jocko.Dial("tcp", c.Addr().String())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// TODO: close the connection?

	resp, err := conn.CreateTopics(&protocol.CreateTopicRequests{
		Requests: []*protocol.CreateTopicRequest{{
			Topic:             topic,
			NumPartitions:     numOfPartitions,
			ReplicationFactor: 1,
		}},
	})
	if err != nil {
		return nil, err
	}
	for _, topicErrCode := range resp.TopicErrorCodes {
		if topicErrCode.ErrorCode != protocol.ErrNone.Code() && topicErrCode.ErrorCode != protocol.ErrTopicAlreadyExists.Code() {
			err := protocol.Errs[topicErrCode.ErrorCode]
			return nil, err
		}
	}

	// TODO: delete
	/*
		mustEncode := func(e protocol.Encoder) []byte {
			var b []byte
			var err error
			if b, err = protocol.Encode(e); err != nil {
				panic(err)
			}
			return b
		}

		req := &protocol.ProduceRequest{
			Timeout: 100 * time.Millisecond,
			TopicData: []*protocol.TopicData{{
				Topic: topic,
				Data: []*protocol.Data{{
					RecordSet: mustEncode(&protocol.MessageSet{
						Offset:   0,
						Messages: []*protocol.Message{{Value: []byte("The message.")}},
					})},
				}}},
		}

		_, err = conn.Produce(req)
		if err != nil {
			t.Fatal(err)
		}
	*/
	// TODO: /delete

	return &MockBroker{
		Configuration: Configuration{
			Address:      c.Addr().String(),
			Topic:        topic,
			Group:        "",
			Enabled:      true,
			OrgWhitelist: nil,
		},
		Connection: conn,
		broker:     c,
	}, nil
}

// Close closes broker
func (mb *MockBroker) Close() {
	// mb.broker.Shutdown()
}
