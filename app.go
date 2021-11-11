package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/apache/plc4x/plc4go/pkg/plc4go"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/drivers"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/transports"

	"os"

	"github.com/go-co-op/gocron"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	// PLC connection string
	var plcConStr = getenv("PLC_ADDRESS", "s7://192.168.167.210/0/0")
	var host = getenv("MQTT_HOST", "tcp://mqtt.eclipseprojects.io:1883")
	var username = getenv("MQTT_USER", "")
	var password = getenv("MQTT_PASSWORD", "")
	var frequencySeconds, _ = strconv.Atoi(getenv("FREQUENCY", "5"))
	var parametersString = getenv("PARAMETERS", "{\"motor-current\": \"%DB444.DBD8:REAL\", \"position\": \"%DB444.DBD0:REAL\", \"rand_val\": \"%DB444.DBD4:REAL\"}")

	var parameters map[string]string

	err := json.Unmarshal([]byte(parametersString), &parameters)
	if err != nil {
		return
	}

	c := connectMQTT(host, username, password)
	defer disconnectMQTT(c)
	readPlc(plcConStr, frequencySeconds, parameters, c)
}

//define a function for the default message handler
var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func connectMQTT(host string, username string, password string) MQTT.Client {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(host)
	clientId := "portal-connect-" + strconv.Itoa(rand.Int())
	fmt.Println("Client ID:", clientId)
	opts.SetClientID(clientId)
	opts.SetDefaultPublishHandler(f)

	if username != "" {
		opts.SetUsername(username)
		opts.SetPassword(password)
	}

	//create and start a client using the above ClientOptions
	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//subscribe to the topic /portal-test/sample and request messages to be delivered
	//at a maximum qos of zero, wait for the receipt to confirm the subscription
	if token := c.Subscribe("portal-test/sample", 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	return c
}

func disconnectMQTT(c MQTT.Client) {
	//unsubscribe from /portal-test/sample
	if token := c.Unsubscribe("portal-test/sample"); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	c.Disconnect(250)
}

func publishMQTT(c MQTT.Client, label string, value string) {
	text := fmt.Sprintf("key: %s, value: %s", label, value)
	token := c.Publish("portal-test/sample", 0, false, text)
	token.Wait()
}

func publishMapMQTT(c MQTT.Client, values map[string]interface{}) {
	jsonString, _ := json.Marshal(values)
	fmt.Println("Sending", jsonString)
	token := c.Publish("portal-test/sample", 0, false, jsonString)
	token.Wait()
}

func readPlc(plcConStr string, seconds int, parameters map[string]string, c MQTT.Client) {
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// Follow this: http://plc4x.apache.org/users/getting-started/plc4go.html
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	s := gocron.NewScheduler(time.UTC)

	p, _ := NewPool(plcConStr)

	// Schedule the Event
	_, err := s.Every(seconds).Seconds().SingletonMode().Do(performRequest, c, p, parameters)
	if err != nil {
		return
	}

	// DO a regular reconnect
	//_, err = s.Every(1).Minutes().Do(performReconnect, p)

	// Close the App
	_, err = s.Every(10).Minutes().WaitForSchedule().Do(closeApp)

	// Run the main "event loop"
	s.StartBlocking()
}

func closeApp() {
	fmt.Println("Force Close...")
	os.Exit(1)
}

func performReconnect(p Pool) {
	fmt.Println("I will just reset the connection now!")
	err := p.Reconnect()
	if err != nil {
		fmt.Println("Issue while reconnecting...")
		return
	}
}

type DefaultPool struct {
	Pool
	connectionString  string
	driverManager     plc4go.PlcDriverManager
	connectionChannel <-chan plc4go.PlcConnection
	connection        plc4go.PlcConnection
}

type Pool interface {
	Get() plc4go.PlcConnection
	Reconnect() error
}

func (p *DefaultPool) Reconnect() error {
	// Close
	closeChan := p.connection.Close()

	// Wait to close
	<-closeChan

	// Get a connection to a remote PLC
	connectionRequestChanel := p.driverManager.GetConnection(p.connectionString)

	// Wait for the driver to connect (or not)
	connectionResult := <-connectionRequestChanel

	// Check if something went wrong
	if connectionResult.Err != nil {
		fmt.Printf("Error connecting to PLC: %s", connectionResult.Err.Error())
		return connectionResult.Err
	}

	// If all was ok, get the connection instance
	connection := connectionResult.Connection
	p.connection = connection

	return nil
}

func NewPool(plcConStr string) (Pool, error) {
	// Create a new instance of the PlcDriverManager
	driverManager := plc4go.NewPlcDriverManager()

	connectionChannel := make(chan plc4go.PlcConnection)

	go func() {
		// Register the Transports
		transports.RegisterTcpTransport(driverManager)
		transports.RegisterUdpTransport(driverManager)

		// Register the Drivers
		drivers.RegisterS7Driver(driverManager)

		// Get a connection to a remote PLC
		connectionRequestChanel := driverManager.GetConnection(plcConStr)

		// Wait for the driver to connect (or not)
		connectionResult := <-connectionRequestChanel

		// Check if something went wrong
		if connectionResult.Err != nil {
			fmt.Printf("Error connecting to PLC: %s", connectionResult.Err.Error())
		} else {
			connectionChannel <- connectionResult.Connection
		}
	}()

	return &DefaultPool{driverManager: driverManager, connectionString: plcConStr, connection: nil, connectionChannel: connectionChannel}, nil
}

func (p *DefaultPool) Get() plc4go.PlcConnection {
	if p.connection == nil {
		conn := <-p.connectionChannel
		p.connection = conn
	}
	return p.connection
}

func performRequest(c MQTT.Client, pool Pool, parameters map[string]string) {
	fmt.Println("Doing a Request now...")
	var connection plc4go.PlcConnection

	connection = pool.Get()

	// Prepare a read-request
	builder := connection.ReadRequestBuilder()

	for key, value := range parameters {
		builder.AddQuery(key, value)
	}

	readRequest, err := builder.Build()
	if err != nil {
		return
	}

	// Execute a read-request
	readResponseChanel := readRequest.Execute()

	// Wait for the response to finish
	readRequestResult := <-readResponseChanel
	if readRequestResult.Err != nil {
		fmt.Printf("error executing read-request: %s", readRequestResult.Err.Error())
		return
	}

	// ...
	result := make(map[string]interface{})
	for key := range parameters {
		val := readRequestResult.Response.GetValue(key)
		result[key] = val.GetString()
	}

	publishMapMQTT(c, result)
}
