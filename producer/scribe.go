// Copyright 2015 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package producer

import (
	"github.com/artyom/scribe"
	"github.com/artyom/thrift"
	"github.com/trivago/gollum/core"
	"github.com/trivago/gollum/core/log"
	"github.com/trivago/gollum/shared"
	"sync"
	"sync/atomic"
	"time"
)

// Scribe producer plugin
// Configuration example
//
//   - "producer.Scribe":
//     Enable: true
//     Address: "localhost:1463"
//     ConnectionBufferSizeKB: 1024
//     BatchMaxCount: 8192
//     BatchFlushCount: 4096
//     BatchTimeoutSec: 5
//     Category:
//       "console" : "console"
//       "_GOLLUM_"  : "_GOLLUM_"
//     Stream:
//       - "console"
//       - "_GOLLUM_"
//
// The scribe producer allows sending messages to Facebook's scribe.
// This producer uses a fuse breaker if the connection to the scribe server is
// lost.
//
// Address defines the host and port to connect to.
// By default this is set to "localhost:1463".
//
// ConnectionBufferSizeKB sets the connection buffer size in KB. By default this
// is set to 1024, i.e. 1 MB buffer.
//
// BatchMaxCount defines the maximum number of messages that can be buffered
// before a flush is mandatory. If the buffer is full and a flush is still
// underway or cannot be triggered out of other reasons, the producer will
// block. By default this is set to 8192.
//
// BatchFlushCount defines the number of messages to be buffered before they are
// written to disk. This setting is clamped to BatchMaxCount.
// By default this is set to BatchMaxCount / 2.
//
// BatchTimeoutSec defines the maximum number of seconds to wait after the last
// message arrived before a batch is flushed automatically. By default this is
// set to 5.
//
// Category maps a stream to a specific scribe category. You can define the
// wildcard stream (*) here, too. When set, all streams that do not have a
// specific mapping will go to this category (including _GOLLUM_).
// If no category mappings are set the stream name is used.
type Scribe struct {
	core.ProducerBase
	scribe           *scribe.ScribeClient
	transport        *thrift.TFramedTransport
	socket           *thrift.TSocket
	category         map[core.MessageStreamID]string
	batch            core.MessageBatch
	batchTimeout     time.Duration
	batchMaxCount    int
	batchFlushCount  int
	bufferSizeByte   int
	windowSize       int
	counters         map[string]*int64
	lastMetricUpdate time.Time
}

const (
	scribeMetricMessages    = "Scribe:Messages-"
	scribeMetricMessagesSec = "Scribe:MessagesSec-"
	scribeMetricWindowSize  = "Scribe:WindowSize"
	scribeMaxRetries        = 30
	scribeMaxSleepTimeMs    = 3000
)

func init() {
	shared.TypeRegistry.Register(Scribe{})
}

// Configure initializes this producer with values from a plugin config.
func (prod *Scribe) Configure(conf core.PluginConfig) error {
	err := prod.ProducerBase.Configure(conf)
	if err != nil {
		return err
	}
	prod.SetStopCallback(prod.close)
	host := conf.GetString("Address", "localhost:1463")

	prod.batchMaxCount = conf.GetInt("BatchMaxCount", 8192)
	prod.windowSize = prod.batchMaxCount
	prod.batchFlushCount = conf.GetInt("BatchFlushCount", prod.batchMaxCount/2)
	prod.batchFlushCount = shared.MinI(prod.batchFlushCount, prod.batchMaxCount)
	prod.batchTimeout = time.Duration(conf.GetInt("BatchTimeoutSec", 5)) * time.Second
	prod.batch = core.NewMessageBatch(prod.batchMaxCount)

	prod.bufferSizeByte = conf.GetInt("ConnectionBufferSizeKB", 1<<10) << 10 // 1 MB
	prod.category = conf.GetStreamMap("Category", "")

	// Initialize scribe connection

	prod.socket, err = thrift.NewTSocket(host)
	if err != nil {
		Log.Error.Print("Scribe socket error:", err)
		return err
	}

	prod.transport = thrift.NewTFramedTransport(prod.socket)
	binProtocol := thrift.NewTBinaryProtocol(prod.transport, false, false)
	prod.scribe = scribe.NewScribeClientProtocol(prod.transport, binProtocol, binProtocol)
	prod.lastMetricUpdate = time.Now()
	prod.counters = make(map[string]*int64)

	shared.Metric.New(scribeMetricWindowSize)
	shared.Metric.SetI(scribeMetricWindowSize, prod.windowSize)

	for _, category := range prod.category {
		shared.Metric.New(scribeMetricMessages + category)
		shared.Metric.New(scribeMetricMessagesSec + category)
		prod.counters[category] = new(int64)
	}

	prod.SetCheckFuseCallback(prod.tryOpenConnection)
	return nil
}

func (prod *Scribe) bufferMessage(msg core.Message) {
	prod.batch.AppendOrFlush(msg, prod.sendBatch, prod.IsActiveOrStopping, prod.Drop)
}

func (prod *Scribe) sendBatchOnTimeOut() {
	// Update metrics
	duration := time.Since(prod.lastMetricUpdate)
	prod.lastMetricUpdate = time.Now()

	for category, counter := range prod.counters {
		count := atomic.SwapInt64(counter, 0)
		shared.Metric.Add(scribeMetricMessages+category, count)
		shared.Metric.SetF(scribeMetricMessagesSec+category, float64(count)/duration.Seconds())
	}

	// Flush if necessary
	if prod.batch.ReachedTimeThreshold(prod.batchTimeout) || prod.batch.ReachedSizeThreshold(prod.batchFlushCount) {
		prod.sendBatch()
	}
}

func (prod *Scribe) tryOpenConnection() bool {
	if prod.transport.IsOpen() {
		return true
	}

	err := prod.transport.Open()
	if err != nil {
		Log.Error.Print("Scribe connection error:", err)
		return false
	}

	prod.socket.Conn().(bufferedConn).SetWriteBuffer(prod.bufferSizeByte)
	prod.Control() <- core.PluginControlFuseActive
	return true
}

func (prod *Scribe) sendBatch() {
	if prod.tryOpenConnection() {
		prod.batch.Flush(prod.transformMessages)
	} else if prod.IsStopping() {
		prod.batch.Flush(prod.dropMessages)
	}
}

func (prod *Scribe) dropMessages(messages []core.Message) {
	for _, msg := range messages {
		prod.Drop(msg)
	}
}

func (prod *Scribe) transformMessages(messages []core.Message) {
	logBuffer := make([]*scribe.LogEntry, len(messages))

	for idx, msg := range messages {
		msg.Data, msg.StreamID = prod.Format(msg)

		category, exists := prod.category[msg.StreamID]
		if !exists {
			if category, exists = prod.category[core.WildcardStreamID]; !exists {
				category = core.StreamRegistry.GetStreamName(msg.StreamID)
			}
			shared.Metric.New(scribeMetricMessages + category)
			shared.Metric.New(scribeMetricMessagesSec + category)
			prod.counters[category] = new(int64)
			prod.category[msg.StreamID] = category
		}

		logBuffer[idx] = &scribe.LogEntry{
			Category: category,
			Message:  string(msg.Data),
		}

		atomic.AddInt64(prod.counters[category], 1)
	}

	// Try to send the whole batch.
	// If this fails, reduce the number of items send until sending succeeds.

	idxStart := 0
	for retryCount := 0; retryCount < scribeMaxRetries; retryCount++ {
		idxEnd := shared.MinI(len(logBuffer), idxStart+prod.windowSize)
		resultCode, err := prod.scribe.Log(logBuffer[idxStart:idxEnd])

		if resultCode == scribe.ResultCode_OK {
			idxStart = idxEnd
			if idxStart < len(logBuffer) {
				retryCount = -1 // incremented to 0 after continue
				continue        // ### continue, data left to send ###
			}
			// Grow the window on success so we don't get stuck at 1
			if prod.windowSize < len(logBuffer) {
				prod.windowSize += (len(logBuffer) - prod.windowSize) / 2
			}
			return // ### return, success ###
		}

		if err != nil || resultCode != scribe.ResultCode_TRY_LATER {
			Log.Error.Printf("Scribe log error %d: %s", resultCode, err.Error())
			prod.transport.Close() // reconnect
			prod.dropMessages(messages[idxStart:])
			return // ### return, failure ###
		}

		prod.windowSize = shared.MaxI(1, prod.windowSize/2)
		shared.Metric.SetI(scribeMetricWindowSize, prod.windowSize)

		time.Sleep(time.Duration(scribeMaxSleepTimeMs/scribeMaxRetries) * time.Millisecond)
	}

	Log.Error.Printf("Scribe server seems to be busy")
	prod.dropMessages(messages[idxStart:])
}

func (prod *Scribe) close() {
	defer func() {
		prod.transport.Close()
		prod.socket.Close()
		prod.WorkerDone()
	}()

	prod.CloseMessageChannel(prod.bufferMessage)
	prod.batch.Close(prod.transformMessages, prod.GetShutdownTimeout())

	if !prod.IsStopping() {
		prod.Control() <- core.PluginControlFuseBurn
	}
}

// Produce writes to a buffer that is sent to scribe.
func (prod *Scribe) Produce(workers *sync.WaitGroup) {
	prod.AddMainWorker(workers)
	prod.TickerMessageControlLoop(prod.bufferMessage, prod.batchTimeout, prod.sendBatchOnTimeOut)
}
