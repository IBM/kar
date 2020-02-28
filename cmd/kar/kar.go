package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/Shopify/sarama"
	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.ibm.com/solsa/kar.git/pkg/logger"
)

var (
	appName = flag.String("app", "", "The name of the application")

	serviceName = flag.String("service", "", "The name of the service being joined to the application")
	servicePort = flag.Int("port", 3000, "The HTTP port for the service")
	serviceURL  string

	karPort = flag.Int("listen", 0, "The HTTP port for KAR to listen on") // defaults to 0 for dynamic selection

	// kafka
	kafkaBrokers  = flag.String("kafka_brokers", "", "The Kafka brokers to connect to, as a comma separated list")
	kafkaTLS      = flag.Bool("kafka_tls", false, "Use TLS to communicate with Kafka")
	kafkaUser     = flag.String("kafka_user", "", "The SASL username if any")
	kafkaPassword = flag.String("kafka_password", "", "The SASL password if any")
	kafkaVersion  = flag.String("kafka_version", "", "Kafka cluster version")

	kafkaProducer          sarama.SyncProducer
	kafkaPartitionConsumer sarama.PartitionConsumer
	kafkaTopic             string

	// redis
	redisAddress  = flag.String("redis_address", "", "The address of the Redis server to connect to")
	redisTLS      = flag.Bool("redis_tls", false, "Use TLS to communicate with Redis")
	redisPassword = flag.String("redis_password", "", "The password of the Redis server if any")

	redisConnection redis.Conn
	redisLock       sync.Mutex

	// logging
	verbosity = flag.Int("v", 0, "Logging verbosity")

	// pending requests: map uuids to channel (string -> channel string)
	requests = sync.Map{}

	// termination
	quit = make(chan struct{})
	wg   = sync.WaitGroup{}
)

func send(service string, message map[string]string) error {
	msg, err := json.Marshal(message)
	if err != nil {
		logger.Error("failed to marshal message %v: %v", message, err)
		return err
	}

	topic := fmt.Sprintf("kar-%s-%s", *appName, service)

	partition, offset, err := kafkaProducer.SendMessage(&sarama.ProducerMessage{
		// TODO Key?
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	})
	if err != nil {
		logger.Error("failed to send message to topic %s: %v", topic, err)
	}

	logger.Info("sent message on topic %s, at partition %d, offset %d, with value %s", topic, partition, offset, string(msg))
	return nil
}

func post(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")

	buf := bytes.Buffer{}
	buf.ReadFrom(r.Body)

	err := send(service, map[string]string{
		"kind":         "post",
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"payload":      buf.String()})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message to service %s: %v", service, err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, "OK")
	}
}

func call(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	service := ps.ByName("service")

	id := uuid.New().URN()
	ch := make(chan string)
	requests.Store(id, ch)
	defer requests.Delete(id)

	buf := bytes.Buffer{}
	buf.ReadFrom(r.Body)

	err := send(service, map[string]string{
		"kind":         "call",
		"path":         ps.ByName("path"),
		"content-type": r.Header.Get("Content-Type"),
		"accept":       r.Header.Get("Accept"),
		"origin":       *serviceName,
		"id":           id,
		"payload":      buf.String()})
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to send message to service %s: %v", service, err), http.StatusInternalServerError)
		return
	}

	select {
	case v := <-ch:
		fmt.Fprint(w, v)
	case _, _ = <-quit:
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
	}
}

func reply(m map[string]string, buf bytes.Buffer) {
	err := send(m["origin"], map[string]string{
		"kind":    "reply",
		"id":      m["id"],
		"payload": buf.String()})
	if err != nil {
		logger.Error("failed to reply to request %s from service %s: %v", m["id"], m["origin"], err)
	}
}

func subscriber() {
	defer wg.Done()

	channel := kafkaPartitionConsumer.Messages()
	for {
		select {
		case _, _ = <-quit:
			return

		case msg := <-channel:
			logger.Info("received message on topic %s, at partition %d, offset %d, with value %s", kafkaTopic, 0, msg.Offset, msg.Value)
			var m map[string]string
			err := json.Unmarshal(msg.Value, &m)
			if err != nil {
				logger.Error("ignoring invalid message from topic %s, at partition %d, offset %d: %v", msg.Topic, msg.Partition, msg.Offset, err)
				continue
			}
			switch m["kind"] {
			case "post":
				_, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"])) // TODO Accept header
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
				}

			case "call":
				res, err := http.Post(serviceURL+m["path"], m["content-type"], strings.NewReader(m["payload"]))
				buf := bytes.Buffer{}
				if err != nil {
					logger.Error("failed to post to %s%s: %v", serviceURL, m["path"], err)
				} else {
					buf.ReadFrom(res.Body)
				}
				reply(m, buf)

			case "reply":
				ch, _ := requests.Load(m["id"])
				ch.(chan string) <- m["payload"]

			default:
				logger.Error("failed to process message with kind %s, from topic %s, at partition %d, offset %d", m["kind"], msg.Topic, msg.Partition, msg.Offset)
			}
		}
	}
}

func setKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := fmt.Sprintf("%s-%s", *appName, ps.ByName("key"))

	buf := bytes.Buffer{}
	buf.ReadFrom(r.Body)

	redisLock.Lock()
	reply, err := redisConnection.Do("SET", key, buf.String())
	redisLock.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to set key %s: %v", key, err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, reply)
	}
}

func getKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := fmt.Sprintf("%s-%s", *appName, ps.ByName("key"))

	buf := bytes.Buffer{}
	buf.ReadFrom(r.Body)

	redisLock.Lock()
	reply, err := redisConnection.Do("GET", key)
	redisLock.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get key %s: %v", key, err), http.StatusInternalServerError)
	} else if reply != nil {
		fmt.Fprintln(w, string(reply.([]byte)))
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func delKey(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	key := fmt.Sprintf("%s-%s", *appName, ps.ByName("key"))

	redisLock.Lock()
	reply, err := redisConnection.Do("DEL", key)
	redisLock.Unlock()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete key %s: %v", key, err), http.StatusInternalServerError)
	} else {
		fmt.Fprintln(w, strconv.FormatInt(reply.(int64), 10))
	}
}

func server(listener net.Listener) {
	defer wg.Done()

	router := httprouter.New()

	router.POST("/kar/post/:service/*path", post)
	router.POST("/kar/call/:service/*path", call)

	router.POST("/kar/set/:key", setKey)
	router.GET("/kar/get/:key", getKey)
	router.GET("/kar/del/:key", delKey)

	srv := &http.Server{Handler: router}

	go func() {
		if err := srv.Serve(listener); err != http.ErrServerClosed {
			logger.Fatal("HTTP server failed: %v", err)
		}
	}()

	_, _ = <-quit
	if err := srv.Shutdown(context.Background()); err != nil {
		logger.Fatal("failed to shutdown HTTP server: %v", err)
	}
}

func dump(prefix string, in io.Reader) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		log.Printf(prefix+"%s", scanner.Text())
	}
}

func main() {
	flag.Parse()
	logger.SetVerbosity(logger.Severity(*verbosity))

	logger.Warning("starting...")

	if *appName == "" {
		logger.Fatal("app name is required")
	}

	if *serviceName == "" {
		logger.Fatal("service name is required")
	}

	if !*kafkaTLS {
		*kafkaTLS, _ = strconv.ParseBool(os.Getenv("KAFKA_TLS"))
	}

	if *kafkaBrokers == "" {
		*kafkaBrokers = os.Getenv("KAFKA_BROKERS")
		if *kafkaBrokers == "" {
			logger.Fatal("at least one Kafka broker is required")
		}
	}

	if *kafkaUser == "" {
		*kafkaUser = os.Getenv("KAFKA_USER")
		if *kafkaUser == "" {
			*kafkaUser = "token"
		}
	}

	if *kafkaPassword == "" {
		*kafkaPassword = os.Getenv("KAFKA_PASSWORD")
	}

	if *kafkaVersion == "" {
		*kafkaVersion = os.Getenv("KAFKA_VERSION")
		if *kafkaVersion == "" {
			*kafkaVersion = "2.2.0"
		}
	}

	if !*redisTLS {
		*redisTLS, _ = strconv.ParseBool(os.Getenv("REDIS_TLS"))
	}

	if *redisAddress == "" {
		*redisAddress = os.Getenv("REDIS_ADDRESS")
		if *redisAddress == "" {
			logger.Fatal("Redis address is required")
		}
	}

	if *redisPassword == "" {
		*redisPassword = os.Getenv("REDIS_PASSWORD")
	}

	version, err := sarama.ParseKafkaVersion(*kafkaVersion)
	if err != nil {
		logger.Fatal("invalid Kafka version: %v", err)
	}

	brokers := strings.Split(*kafkaBrokers, ",")

	kafkaTopic = fmt.Sprintf("kar-%s-%s", *appName, *serviceName)

	serviceURL = fmt.Sprintf("http://127.0.0.1:%d", *servicePort)

	conf := sarama.NewConfig()
	conf.Version = version
	conf.ClientID = "kar"
	conf.Producer.Return.Successes = true
	conf.Producer.RequiredAcks = sarama.WaitForAll

	if *kafkaPassword != "" {
		conf.Net.SASL.Enable = true
		conf.Net.SASL.User = *kafkaUser
		conf.Net.SASL.Password = *kafkaPassword
		conf.Net.SASL.Handshake = true
		conf.Net.SASL.Mechanism = sarama.SASLTypePlaintext
	}

	if *kafkaTLS {
		conf.Net.TLS.Enable = true
		conf.Net.TLS.Config = &tls.Config{
			InsecureSkipVerify: true, // TODO
		}
	}

	kafkaClusterAdmin, err := sarama.NewClusterAdmin(brokers, conf)
	defer kafkaClusterAdmin.Close()
	if err != nil {
		logger.Fatal("failed to create Kafka cluster admin: %v", err)
	}

	topics, err := kafkaClusterAdmin.ListTopics()
	if err != nil {
		logger.Fatal("failed to list Kafka topics: %v", err)
	}
	if _, ok := topics[kafkaTopic]; !ok {
		err = kafkaClusterAdmin.CreateTopic(kafkaTopic, &sarama.TopicDetail{NumPartitions: 1, ReplicationFactor: 1}, false)
		if err != nil {
			logger.Fatal("failed to create Kafka topic: %v", err.Error())
		}
	}

	kafkaProducer, err = sarama.NewSyncProducer(brokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka producer: %v", err)
	}
	defer kafkaProducer.Close()

	kafkaConsumer, err := sarama.NewConsumer(brokers, conf)
	if err != nil {
		logger.Fatal("failed to create Kafka consumer: %v", err)
	}
	defer kafkaConsumer.Close()

	kafkaPartitionConsumer, err = kafkaConsumer.ConsumePartition(kafkaTopic, 0, sarama.OffsetNewest) // TODO consumer group
	if err != nil {
		logger.Fatal("failed to create Kafka partition consumer: %v", err)
	}
	defer kafkaPartitionConsumer.Close()

	redisOptions := []redis.DialOption{}
	if *redisTLS {
		redisOptions = append(redisOptions, redis.DialUseTLS(true))
		redisOptions = append(redisOptions, redis.DialTLSSkipVerify(true)) // TODO
	}
	if *redisPassword != "" {
		redisOptions = append(redisOptions, redis.DialPassword(*redisPassword))
	}

	redisConnection, err = redis.Dial("tcp", *redisAddress, redisOptions...)
	if err != nil {
		logger.Fatal("failed to connect to Redis: %v", err)
	}
	defer redisConnection.Close()

	wg.Add(1)
	go subscriber()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", *karPort))
	if err != nil {
		logger.Fatal("Listener failed: %v", err)
	}

	wg.Add(1)
	go server(listener)

	args := flag.Args()

	port1 := fmt.Sprintf("KAR_PORT=%d", listener.Addr().(*net.TCPAddr).Port)
	port2 := fmt.Sprintf("KAR_APP_PORT=%d", *servicePort)
	logger.Info("%s, %s", port1, port2)

	if len(args) > 0 {
		logger.Info("launching service...")

		cmd := exec.Command(args[0], args[1:]...)
		cmd.Env = append(os.Environ(), port1, port2)
		cmd.Stdin = os.Stdin
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			logger.Error("failed to capture stdout from service: %v", err)
		}
		go dump("[STDOUT] ", stdout)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			logger.Error("failed to capture stderr from service: %v", err)
		}
		go dump("[STDERR] ", stderr)

		if err := cmd.Start(); err != nil {
			logger.Error("failed to start service: %v", err)
		}

		if err := cmd.Wait(); err != nil {
			if v, ok := err.(*exec.ExitError); ok {
				logger.Info("service exited with status code %d", v.ExitCode())
			} else {
				logger.Fatal("error waiting for service: %v", err)
			}
		} else {
			logger.Info("service exited normally")
		}

		close(quit)
	}

	wg.Wait()

	logger.Warning("exiting...")
}
