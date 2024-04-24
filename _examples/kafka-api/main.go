package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/IBM/sarama"
	"github.com/kataras/iris/v12"
)

/*
First of all, read about Apache Kafka, install and run it, if you didn't already: https://kafka.apache.org/quickstart

Secondly, install your favourite Go library for Apache Kafka communication.
I have chosen the shopify's one although I really loved the `segmentio/kafka-go` as well but it needs more to be done there
and you will be bored to read all the necessary code required to get started with it, so:
	$ go get -u github.com/IBM/sarama

The minimum Apache Kafka broker(s) version required is 0.10.0.0 but 0.11.x+ is recommended (tested with 2.5.0).

Resources:
	- https://github.com/apache/kafka
	- https://github.com/IBM/sarama/blob/master/examples/http_server/http_server.go
	- DIY
*/

// package-level variables for the sake of the example
// but you can define them inside your main func
// and pass around this config whenever you need to create a client or a producer or a consumer or use a cluster.
var (
	// The Kafka brokers to connect to, as a comma separated list.
	brokers = []string{getenv("KAFKA_1", "localhost:9092")}
	// The config which makes our live easier when passing around, it pre-mades a lot of things for us.
	config *sarama.Config
)

func getenv(key string, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return def
}

func init() {
	config = sarama.NewConfig()
	config.ClientID = "iris-example-client"
	config.Version = sarama.V0_11_0_2
	// config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message.
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Flush.Frequency = 500 * time.Millisecond
	config.Producer.Retry.Max = 10 // Retry up to 10 times to produce the message.
	config.Producer.Return.Successes = true

	// for SASL/basic plain text authentication: config.Net.SASL.
	// config.Net.SASL.Enable = true
	// config.Net.SASL.Handshake = false
	// config.Net.SASL.User = "myuser"
	// config.Net.SASL.Password = "mypass"

	config.Consumer.Return.Errors = true
}

func main() {
	app := iris.New()
	app.OnErrorCode(iris.StatusNotFound, handleNotFound)

	v1 := app.Party("/api/v1")
	{
		topicsAPI := v1.Party("/topics")
		{
			topicsAPI.Post("/", postTopicsHandler) // create a topic.
			topicsAPI.Get("/", getTopicsHandler)   // list all topics.

			topicsAPI.Post("/{topic}/produce", postTopicProduceHandler)  // store to a topic.
			topicsAPI.Get("/{topic}/consume", getTopicConsumeSSEHandler) // retrieve all messages from a topic.
		}
	}

	app.Get("/", docsHandler)

	app.Logger().Infof("Brokers: %s", strings.Join(brokers, ", "))
	// GET      : http://localhost:8080
	// POST, GET: http://localhost:8080/api/v1/topics
	// POST     : http://localhost:8080/api/v1/topics/{topic}/produce?key=my-key
	// GET      : http://localhost:8080/api/v1/topics/{topic}/consume?partition=0&offset=0
	app.Listen(":8080")
}

// simple use-case, you can use templates and views obviously, see the "_examples/views" examples.
func docsHandler(ctx iris.Context) {
	ctx.ContentType("text/html") // or ctx.HTML(fmt.Sprintf(...))
	ctx.Writef(`<!DOCTYPE html>
	<html>
		<head>
			<style>
				th, td {
					border: 1px solid black;
					padding: 15px;
					text-align: left;
				}
			</style>
		</head>`)
	defer ctx.Writef("</html>")

	ctx.Writef("<body>")
	defer ctx.Writef("</body>")

	ctx.Writef(`
	<table>
		<tr>
			<th>Method</th>
			<th>Path</th>
			<th>Handler</th>
		</tr>
	`)
	defer ctx.Writef(`</table>`)

	registeredRoutes := ctx.Application().GetRoutesReadOnly()
	for _, r := range registeredRoutes {
		if r.Path() == "/" { // don't list the root, current one.
			continue
		}

		ctx.Writef(`
			<tr>
				<td>%s</td>
				<td>%s%s</td>
				<td>%s</td>
			</tr>
		`, r.Method(), ctx.Host(), r.Path(), r.MainHandlerName())
	}
}

type httpError struct {
	Code   int    `json:"code"`
	Reason string `json:"reason,omitempty"`
}

func (h httpError) Error() string {
	return fmt.Sprintf("Status Code: %d\nReason: %s", h.Code, h.Reason)
}

func fail(ctx iris.Context, statusCode int, format string, a ...interface{}) {
	reason := "unspecified"
	if format != "" {
		reason = fmt.Sprintf(format, a...)
	}

	err := httpError{
		Code:   statusCode,
		Reason: reason,
	}

	ctx.StopWithJSON(statusCode, err)
}

func handleNotFound(ctx iris.Context) {
	suggestPaths := ctx.FindClosest(3)
	if len(suggestPaths) == 0 {
		ctx.WriteString("not found")
		return
	}

	ctx.HTML("Did you mean?<ul>")
	for _, s := range suggestPaths {
		ctx.HTML(`<li><a href="%s">%s</a></li>`, s, s)
	}
	ctx.HTML("</ul>")
}

// Topic the payload for a kafka topic creation.
type Topic struct {
	Topic             string `json:"topic"`
	Partitions        int32  `json:"partitions"`
	ReplicationFactor int16  `json:"replication"`
	Configs           []kv   `json:"configs,omitempty"`
}

type kv struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func createKafkaTopic(t Topic) error {
	cluster, err := sarama.NewClusterAdmin(brokers, config)
	if err != nil {
		return err
	}
	defer cluster.Close()

	topicName := t.Topic
	topicDetail := sarama.TopicDetail{
		NumPartitions:     t.Partitions,
		ReplicationFactor: t.ReplicationFactor,
	}

	if len(t.Configs) > 0 {
		topicDetail.ConfigEntries = make(map[string]*string, len(t.Configs))
		for _, c := range t.Configs {
			topicDetail.ConfigEntries[c.Key] = &c.Value // generate a ptr, or fill a new(string) with it and use that.
		}
	}

	return cluster.CreateTopic(topicName, &topicDetail, false)
}

func postTopicsHandler(ctx iris.Context) {
	var t Topic
	err := ctx.ReadJSON(&t)
	if err != nil {
		fail(ctx, iris.StatusBadRequest,
			"received invalid topic payload: %v", err)
		return
	}

	// try to create the topic inside kafka.
	err = createKafkaTopic(t)
	if err != nil {
		fail(ctx, iris.StatusInternalServerError,
			"unable to create topic: %v", err)
		return
	}

	ctx.StatusCode(iris.StatusCreated)
	ctx.Writef("Topic %q created", t.Topic)
}

func getKafkaTopics() ([]string, error) {
	client, err := sarama.NewClient(brokers, config)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	return client.Topics()
}

func getTopicsHandler(ctx iris.Context) {
	topics, err := getKafkaTopics()
	if err != nil {
		fail(ctx, iris.StatusInternalServerError,
			"unable to retrieve topics: %v", err)
		return
	}

	ctx.JSON(topics)
}

func produceKafkaMessage(toTopic string, key string, value []byte) (partition int32, offset int64, err error) {
	// On the broker side, you may want to change the following settings to get
	// stronger consistency guarantees:
	// - For your broker, set `unclean.leader.election.enable` to false
	// - For the topic, you could increase `min.insync.replicas`.

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return -1, -1, err
	}
	defer producer.Close()

	// We are not setting a message key, which means that all messages will
	// be distributed randomly over the different partitions.
	return producer.SendMessage(&sarama.ProducerMessage{
		Topic: toTopic,
		Key:   sarama.StringEncoder(key),
		Value: sarama.ByteEncoder(value),
	})
}

func postTopicProduceHandler(ctx iris.Context) {
	topicName := ctx.Params().Get("topic")
	key := ctx.URLParamDefault("key", "default")

	// read the request data and store them as they are (not recommended in production ofcourse, do your own checks here).
	body, err := ctx.GetBody()
	if err != nil {
		fail(ctx, iris.StatusUnprocessableEntity, "unable to read your data: %v", err)
		return
	}

	partition, offset, err := produceKafkaMessage(topicName, key, body)
	if err != nil {
		fail(ctx, iris.StatusInternalServerError, "failed to store your data: %v", err)
		return
	}

	// The tuple (topic, partition, offset) can be used as a unique identifier
	// for a message in a Kafka cluster.
	ctx.Writef("Your data is stored with unique identifier: %s/%d/%d", topicName, partition, offset)
}

type message struct {
	Time time.Time `json:"time"`
	Key  string    `json:"key"`
	// Value []byte/json.RawMessage(if you are sure that you are sending only JSON)    `json:"value"`
	// or:
	Value string `json:"value"` // for simple key-value storage.
}

func getTopicConsumeSSEHandler(ctx iris.Context) {
	flusher, ok := ctx.ResponseWriter().Flusher()
	if !ok {
		ctx.StopWithText(iris.StatusHTTPVersionNotSupported, "streaming unsupported")
		return
	}

	ctx.ContentType("application/json, text/event-stream")
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")

	master, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		fail(ctx, iris.StatusInternalServerError, "unable to start master consumer: %v", err)
		return
	}

	fromTopic := ctx.Params().Get("topic")
	// take the partition, defaults to the first found if not url query parameter "partition" passed.
	var partition int32
	partitions, err := master.Partitions(fromTopic)
	if err != nil {
		master.Close()
		fail(ctx, iris.StatusInternalServerError, "unable to get partitions for topic: '%s': %v", fromTopic, err)
		return
	}

	if len(partitions) > 0 {
		partition = partitions[0]
	}

	partition = ctx.URLParamInt32Default("partition", partition)
	offset := ctx.URLParamInt64Default("offset", sarama.OffsetOldest)

	consumer, err := master.ConsumePartition(fromTopic, partition, offset)
	if err != nil {
		ctx.Application().Logger().Error(err)
		master.Close() // close the master here to avoid any leaks, we will exit.
		fail(ctx, iris.StatusInternalServerError, "unable to start partition consumer: %v", err)
		return
	}

	// `OnClose` fires when the request is finally done (all data read and handler exits) or interrupted by the user.
	ctx.OnClose(func(_ iris.Context) {
		ctx.Application().Logger().Warnf("a client left")

		// Close shuts down the consumer. It must be called after all child
		// PartitionConsumers have already been closed. <-- That is what
		// godocs says but it doesn't work like this.
		// if err = consumer.Close(); err != nil {
		// 	ctx.Application().Logger().Errorf("[%s] unable to close partition consumer: %v", ctx.RemoteAddr(), err)
		// }
		// so close the master only and omit the first ^ consumer.Close:
		if err = master.Close(); err != nil {
			ctx.Application().Logger().Errorf("[%s] unable to close master consumer: %v", ctx.RemoteAddr(), err)
		}
	})

	for {
		select {
		case consumerErr, ok := <-consumer.Errors():
			if !ok {
				return
			}
			ctx.Writef("data: error: {\"reason\": \"%s\"}\n\n", consumerErr.Error())
			flusher.Flush()
		case incoming, ok := <-consumer.Messages():
			if !ok {
				return
			}

			msg := message{
				Time:  incoming.Timestamp,
				Key:   string(incoming.Key),
				Value: string(incoming.Value),
			}

			b, err := json.Marshal(msg)
			if err != nil {
				ctx.Application().Logger().Error(err)
				continue
			}

			ctx.Writef("data: %s\n\n", b)
			flusher.Flush()
		}
	}
}
