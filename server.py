from __future__ import print_function, unicode_literals

import json
import threading
import uuid
from time import sleep
import os

import requests as requests
from proton import Message, Sender, Terminus
from proton._reactor import SenderOption
from proton.handlers import MessagingHandler
from proton.reactor import Container
from requests.auth import HTTPBasicAuth
from dotenv import load_dotenv


class PortalConnectServer(MessagingHandler):
    def __init__(self, server, tenant, deviceId, plcStr):
        super(PortalConnectServer, self).__init__()
        self.deviceId = deviceId
        self.tenant = tenant
        self.server = server
        self.plcStr = plcStr
        self.ready = False
        self.reply_to = str(uuid.uuid4())

    class SessionEnd(SenderOption):

        def apply(self, sender):
            sender: Sender
            sender.source.expiry_policy = Terminus.EXPIRE_WITH_SESSION
            sender.source.timeout = 0
            sender.source.durability = Terminus.NONDURABLE

    def on_start(self, event):
        conn = event.container.connect(url=self.server, user="consumer@HONO", password="verysecret", sasl_enabled=True,
                                       allowed_mechs="PLAIN", allow_insecure_mechs=True)

        event.container.create_receiver(conn, f"telemetry/{self.tenant}")
        event.container.create_receiver(conn, f"event/{self.tenant}")
        event.container.create_receiver(conn, f"command_response/{self.tenant}/{self.reply_to}")
        self.sender = event.container.create_sender(conn, target=f"command/{self.tenant}",
                                                    options=[PortalConnectServer.SessionEnd()])

        def send_commands():
            while True:
                address = f"command/{self.tenant}/{self.deviceId}"
                print(f"Sending command: {address}")
                # message = Message(address=address, subject="work", body="I am the body")
                # self.sender.send(message)

                m2 = Message(address=address, subject="work", body=json.dumps({"host": self.plcStr,"plcField":"%DB444:6.0:REAL"}),
                             reply_to=f"command_response/{self.tenant}/{self.reply_to}", correlation_id= str(uuid.uuid4()))
                self.sender.send(m2)

                sleep(1)

        print("Starting Sender...")
        thread = threading.Thread(target=send_commands, daemon=True)
        thread.start()

    def on_message(self, event):
        self.accept(event.delivery)
        print(f"Received Message at {event.message}")
        if "ttd" in event.message.properties:
            print(f"I got a telemetry message, sending command...")
            address = f"command/{self.tenant}/{self.deviceId}"
            message = Message(address=address, subject="work", body="I am the body")
            self.sender.send(message)

            message = Message(address=address, subject="work", body="Please reply...",
                              properties={"reply-to": self.reply_to})
            self.sender.send(message)

    def on_transport_error(self, event):
        print(f"Transport error: {event}")

    def on_sendable(self, event):
        print("We can send now...")

    def on_rejected(self, event):
        print(f"Message was rejected: {event}")

    def on_released(self, event):
        print(f"Message weas released: {event}")

    def on_accepted(self, event):
        print(f"Message weas acceptefd: {event}")

    def on_settled(self, event):
        print(f"Message weas settled: {event}")


if __name__ == '__main__':
    load_dotenv()

    mqttAdapterIp = os.getenv('HONOMQTTIP') 
    tenantId = os.getenv('HONOTENANT')
    deviceId = os.getenv('HONODEVICE')
    plcStr = os.getenv('PLCURL')
    

    print("Receiver")
    container = Container(PortalConnectServer(f"tcp://{mqttAdapterIp}:1883", tenantId, deviceId, plcStr))

    container.run()

    sleep(500)