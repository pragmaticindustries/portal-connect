package main

import (
	"fmt"

	"github.com/apache/plc4x/plc4go/pkg/plc4go"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/drivers"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/logging"
	"github.com/apache/plc4x/plc4go/pkg/plc4go/transports"

	"os"

	MQTT "github.com/eclipse/paho.mqtt.golang"

	godotenv "github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	c := connectMQTT()
	//defer disconnectMQTT(c)

	//plcConStr := os.Getenv("PLCURL")
	//readPlc(c, plcConStr)

	// Make a channel to receive messages into infinite loop.
	forever := make(chan bool)

	go publishMQTT(c, fmt.Sprintf("{\"host\": \"s7://192.168.167.210/0/0\",\"plcField\":\"%%DB444:6.0:REAL\", \"reply_to\":\"command_response/%s/%s\"}", os.Getenv("HONOTENANT"), os.Getenv("HONODEVICE")))

	<-forever

}

var f MQTT.MessageHandler = func(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
}

func connectMQTT() MQTT.Client {
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(fmt.Sprintf("tcp://%s:1883", os.Getenv("HONOMQTTIP")))
	opts.SetClientID("portal-connect")
	user := fmt.Sprintf("%s@%s", os.Getenv("HONODEVICE"), os.Getenv("HONOTENANT"))
	opts.SetUsername(user)
	opts.SetPassword(os.Getenv("HONODEVICEPASSWORD"))

	c := MQTT.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	//choke := make(chan [2]string)

	/* opts.SetDefaultPublishHandler(func(client MQTT.Client, msg MQTT.Message) {
		fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", msg.Topic(), msg.Payload())
		choke <- [2]string{msg.Topic(), string(msg.Payload())}
	}) */

	opts.SetDefaultPublishHandler(f)

	if token := c.Subscribe(fmt.Sprintf("command/%s/%s", os.Getenv("HONOTENANT"), os.Getenv("HONODEVICE")), 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := c.Subscribe(fmt.Sprintf("command_response/%s/%s", os.Getenv("HONOTENANT"), os.Getenv("HONODEVICE")), 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := c.Subscribe(fmt.Sprintf("telemetry/%s", os.Getenv("HONOTENANT")), 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	if token := c.Subscribe(fmt.Sprintf("event/%s", os.Getenv("HONOTENANT")), 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	//go subscribeCommand(choke)

	return c
}

func subscribeCommand(choke chan [2]string) {
	incoming := <-choke
	fmt.Printf("RECEIVED TOPIC: %s MESSAGE: %s\n", incoming[0], incoming[1])
}

func commandHandler(client MQTT.Client, msg MQTT.Message) {
	fmt.Printf("* [%s] %s\n", msg.Topic(), string(msg.Payload()))
}

func disconnectMQTT(c MQTT.Client) {
	if token := c.Unsubscribe(fmt.Sprintf("command/%s/%s", os.Getenv("HONOTENANT"), os.Getenv("HONODEVICE"))); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}
	c.Disconnect(250)
}

func publishMQTT(c MQTT.Client, msg string) {
	// send commands
	ch := fmt.Sprintf("command/%s/%s", os.Getenv("HONOTENANT"), os.Getenv("HONODEVICE"))
	fmt.Println(ch)
	token := c.Publish(ch, 0, false, msg)
	token.Wait()
}

func readPlc(c MQTT.Client, plcConStr string) {
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// Follow this: http://plc4x.apache.org/users/getting-started/plc4go.html
	//!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

	logging.WarnLevel()

	// Create a new instance of the PlcDriverManager
	driverManager := plc4go.NewPlcDriverManager()

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
		return
	}

	// If all was ok, get the connection instance
	connection := connectionResult.Connection

	// Make sure the connection is closed at the end
	defer connection.Close()

	// Try to ping the remote device
	pingResultChannel := connection.Ping()

	// Wait for the Ping operation to finsh
	pingResult := <-pingResultChannel
	if pingResult.Err != nil {
		fmt.Printf("Couldn't ping device: %s", pingResult.Err.Error())
		return
	}

	if !connection.GetMetadata().CanRead() {
		fmt.Printf("This connection doesn't support read operations")
		return
	}

	// Prepare a read-request
	readRequest, err := connection.ReadRequestBuilder().
		AddQuery("motor-current", "%DB444.DBD8:REAL").
		AddQuery("position", "%DB444.DBD0:REAL").
		AddQuery("rand_val", "%DB444.DBD4:REAL").
		Build()
	if err != nil {
		fmt.Printf("error preparing read-request: %s", connectionResult.Err.Error())
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

	// Do something with the response
	value1 := readRequestResult.Response.GetValue("motor-current")
	value2 := readRequestResult.Response.GetValue("position")
	value3 := readRequestResult.Response.GetValue("rand_val")
	publishMQTT(c, value1.GetString())
	publishMQTT(c, value2.GetString())
	publishMQTT(c, value3.GetString())
}
