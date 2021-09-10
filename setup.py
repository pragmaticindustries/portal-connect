import json
import requests as requests

import os
from dotenv import load_dotenv

load_dotenv()

devicePassword = os.getenv('HONODEVICEPASSWORD')
registryIp = os.getenv('HONOREGISTRYIP')
mqttAdapterIp = os.getenv('HONOMQTTIP') #mqtt Port 1883

tenant = requests.post(f'http://{registryIp}:28080/v1/tenants').json()
tenantId = tenant["id"]

f = open("./.env", "a")

print(f'Registered tenant {tenantId}')

f.write('\n')
f.write("HONOTENANT=")
f.write(tenantId)
f.write('\n')

# Add Device to Tenant
device = requests.post(f'http://{registryIp}:28080/v1/devices/{tenantId}').json()
deviceId = device["id"]
print(f'Registered device {deviceId}')

f.write("HONODEVICE=")
f.write(deviceId)
f.write('\n')
f.close()

code = requests.put(f'http://{registryIp}:28080/v1/credentials/{tenantId}/{deviceId}',
                        headers={'content-type': 'application/json'},
                        data=json.dumps(
                            [{"type": "hashed-password", "auth-id": deviceId,
                            "secrets": [{"pwd-plain": devicePassword}]}]))

if code.status_code == 204:
    print("Password is set!")
else:
    print("Unnable to set Password")