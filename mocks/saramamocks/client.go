// Code generated by mockery v1.0.0. DO NOT EDIT.

package saramamocks

import (
	sarama "github.com/Shopify/sarama"
	mock "github.com/stretchr/testify/mock"
)

// Client is an autogenerated mock type for the Client type
type Client struct {
	mock.Mock
}

// Broker provides a mock function with given fields: brokerID
func (_m *Client) Broker(brokerID int32) (*sarama.Broker, error) {
	ret := _m.Called(brokerID)

	var r0 *sarama.Broker
	if rf, ok := ret.Get(0).(func(int32) *sarama.Broker); ok {
		r0 = rf(brokerID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Broker)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(int32) error); ok {
		r1 = rf(brokerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Brokers provides a mock function with given fields:
func (_m *Client) Brokers() []*sarama.Broker {
	ret := _m.Called()

	var r0 []*sarama.Broker
	if rf, ok := ret.Get(0).(func() []*sarama.Broker); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*sarama.Broker)
		}
	}

	return r0
}

// Close provides a mock function with given fields:
func (_m *Client) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Closed provides a mock function with given fields:
func (_m *Client) Closed() bool {
	ret := _m.Called()

	var r0 bool
	if rf, ok := ret.Get(0).(func() bool); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(bool)
	}

	return r0
}

// Config provides a mock function with given fields:
func (_m *Client) Config() *sarama.Config {
	ret := _m.Called()

	var r0 *sarama.Config
	if rf, ok := ret.Get(0).(func() *sarama.Config); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Config)
		}
	}

	return r0
}

// Controller provides a mock function with given fields:
func (_m *Client) Controller() (*sarama.Broker, error) {
	ret := _m.Called()

	var r0 *sarama.Broker
	if rf, ok := ret.Get(0).(func() *sarama.Broker); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Broker)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Coordinator provides a mock function with given fields: consumerGroup
func (_m *Client) Coordinator(consumerGroup string) (*sarama.Broker, error) {
	ret := _m.Called(consumerGroup)

	var r0 *sarama.Broker
	if rf, ok := ret.Get(0).(func(string) *sarama.Broker); ok {
		r0 = rf(consumerGroup)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Broker)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(consumerGroup)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetOffset provides a mock function with given fields: topic, partitionID, time
func (_m *Client) GetOffset(topic string, partitionID int32, time int64) (int64, error) {
	ret := _m.Called(topic, partitionID, time)

	var r0 int64
	if rf, ok := ret.Get(0).(func(string, int32, int64) int64); ok {
		r0 = rf(topic, partitionID, time)
	} else {
		r0 = ret.Get(0).(int64)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int32, int64) error); ok {
		r1 = rf(topic, partitionID, time)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InSyncReplicas provides a mock function with given fields: topic, partitionID
func (_m *Client) InSyncReplicas(topic string, partitionID int32) ([]int32, error) {
	ret := _m.Called(topic, partitionID)

	var r0 []int32
	if rf, ok := ret.Get(0).(func(string, int32) []int32); ok {
		r0 = rf(topic, partitionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int32) error); ok {
		r1 = rf(topic, partitionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// InitProducerID provides a mock function with given fields:
func (_m *Client) InitProducerID() (*sarama.InitProducerIDResponse, error) {
	ret := _m.Called()

	var r0 *sarama.InitProducerIDResponse
	if rf, ok := ret.Get(0).(func() *sarama.InitProducerIDResponse); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.InitProducerIDResponse)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Leader provides a mock function with given fields: topic, partitionID
func (_m *Client) Leader(topic string, partitionID int32) (*sarama.Broker, error) {
	ret := _m.Called(topic, partitionID)

	var r0 *sarama.Broker
	if rf, ok := ret.Get(0).(func(string, int32) *sarama.Broker); ok {
		r0 = rf(topic, partitionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Broker)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int32) error); ok {
		r1 = rf(topic, partitionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// OfflineReplicas provides a mock function with given fields: topic, partitionID
func (_m *Client) OfflineReplicas(topic string, partitionID int32) ([]int32, error) {
	ret := _m.Called(topic, partitionID)

	var r0 []int32
	if rf, ok := ret.Get(0).(func(string, int32) []int32); ok {
		r0 = rf(topic, partitionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int32) error); ok {
		r1 = rf(topic, partitionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Partitions provides a mock function with given fields: topic
func (_m *Client) Partitions(topic string) ([]int32, error) {
	ret := _m.Called(topic)

	var r0 []int32
	if rf, ok := ret.Get(0).(func(string) []int32); ok {
		r0 = rf(topic)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(topic)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RefreshBrokers provides a mock function with given fields: addrs
func (_m *Client) RefreshBrokers(addrs []string) error {
	ret := _m.Called(addrs)

	var r0 error
	if rf, ok := ret.Get(0).(func([]string) error); ok {
		r0 = rf(addrs)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RefreshController provides a mock function with given fields:
func (_m *Client) RefreshController() (*sarama.Broker, error) {
	ret := _m.Called()

	var r0 *sarama.Broker
	if rf, ok := ret.Get(0).(func() *sarama.Broker); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*sarama.Broker)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RefreshCoordinator provides a mock function with given fields: consumerGroup
func (_m *Client) RefreshCoordinator(consumerGroup string) error {
	ret := _m.Called(consumerGroup)

	var r0 error
	if rf, ok := ret.Get(0).(func(string) error); ok {
		r0 = rf(consumerGroup)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// RefreshMetadata provides a mock function with given fields: topics
func (_m *Client) RefreshMetadata(topics ...string) error {
	_va := make([]interface{}, len(topics))
	for _i := range topics {
		_va[_i] = topics[_i]
	}
	var _ca []interface{}
	_ca = append(_ca, _va...)
	ret := _m.Called(_ca...)

	var r0 error
	if rf, ok := ret.Get(0).(func(...string) error); ok {
		r0 = rf(topics...)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Replicas provides a mock function with given fields: topic, partitionID
func (_m *Client) Replicas(topic string, partitionID int32) ([]int32, error) {
	ret := _m.Called(topic, partitionID)

	var r0 []int32
	if rf, ok := ret.Get(0).(func(string, int32) []int32); ok {
		r0 = rf(topic, partitionID)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, int32) error); ok {
		r1 = rf(topic, partitionID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Topics provides a mock function with given fields:
func (_m *Client) Topics() ([]string, error) {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func() error); ok {
		r1 = rf()
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// WritablePartitions provides a mock function with given fields: topic
func (_m *Client) WritablePartitions(topic string) ([]int32, error) {
	ret := _m.Called(topic)

	var r0 []int32
	if rf, ok := ret.Get(0).(func(string) []int32); ok {
		r0 = rf(topic)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]int32)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string) error); ok {
		r1 = rf(topic)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}
